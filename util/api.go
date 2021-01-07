package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

// API supply REST/gRPC api common utilities
type API struct {
	// JWT token secret
	TokenSec []byte
	// AuthToken type JWT token string
	Token AuthToken
	// authenticated JWT claims map
	Claims jwt.MapClaims
	// gRPC api list not requir auth check
	NoAuth []string
	Log    *log.Entry
}

// Error is REST api error handling function
// log 1st error message if exist
// report joint 2nd up to the end error messages if exist, otherwise report the same 1st message
// response http error code with json body using key "error"
func (api *API) Error(w http.ResponseWriter, code int, err ...string) {
	if len(err) == 0 {
		err = append(err, "server error")
	}
	api.Log.Error(err[0])
	res := make(map[string]string)
	if len(err) == 1 {
		res["error"] = err[0]
	} else {
		res["error"] = strings.Join(err[1:], ", ")
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(res)
}

// Errpc is gRPC api error handling function
// log 1st error message if exist
// report joint 2nd up to the end error messages if exist, otherwise report the same 1st message
// generate gRPC status message
func (api *API) Errpc(code codes.Code, err ...string) error {
	if len(err) == 0 {
		err = append(err, "server error")
	}
	api.Log.Error(err[0])
	var res string
	if len(err) == 1 {
		res = err[0]
	} else {
		res = strings.Join(err[1:], ", ")
	}
	return status.Errorf(code, res)
}

// Auth http handler function
// perform JWT authentication and pass token to the next handler by context
func (api *API) Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
		if len(authHeader) != 2 {
			api.Error(w, http.StatusUnauthorized, "Malformed token", "Unauthorized")
			return
		}
		jwtToken := authHeader[1]
		token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return api.TokenSec, nil
		})
		if err != nil {
			api.Error(w, http.StatusUnauthorized, fmt.Sprintf("JWT auth fail: %v", err), "Unauthorized")
			return
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			api.Token = AuthToken(r.Header.Get("Authorization"))
			api.Claims = claims
			next(w, r)
		} else {
			api.Error(w, http.StatusUnauthorized, "invalid token claims", "Unauthorized")
		}
		return
	}
}

/*
func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
		if len(authHeader) != 2 {
			log.Warn("Malformed token")
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			jwtToken := authHeader[1]
			token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return TOKENSEC, nil
			})

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				ctx := context.WithValue(r.Context(), "token", claims)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				log.WithError(err).Warnf("Unauthorized Access Atempt, uid: %v", claims["uid"])
				w.WriteHeader(http.StatusUnauthorized)
			}
		}
	})
}*/

// AuthGrpcUnary gRPC handler function, called by gRPC interceptor for api JWT authentication
// perform Unary function JWT authentication and pass token to the next handler by context
func (api *API) AuthGrpcUnary(ctx context.Context, req interface{}, srv *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// skip calls no auth requirement
	for _, a := range api.NoAuth {
		if a == srv.FullMethod {
			return handler(ctx, req)
		}
	}
	// retrieve token from gRPC meta
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, api.Errpc(codes.Unauthenticated, "JWT auth missing metadata", "Unauthorized")
	}
	ts, exist := md["authorization"]
	if !exist {
		ts, exist = md["Authorization"]
		if !exist {
			return nil, api.Errpc(codes.Unauthenticated, "JWT auth missing authorization field in metadata", "Unauthorized")
		}
	}
	token, err := jwt.Parse(strings.TrimPrefix(ts[0], "Bearer "), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return api.TokenSec, nil
	})
	if err != nil {
		return nil, api.Errpc(codes.Unauthenticated, fmt.Sprintf("JWT auth fail: %v", err), "Unauthorized")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		api.Token = AuthToken(ts[0])
		api.Claims = claims
		return handler(ctx, req)
	}
	return nil, api.Errpc(codes.Unauthenticated, fmt.Sprintf("invalid token claims: %v", err), "Unauthorized")
}

// ApiGet pass JWT from original request to target api
// result will be saved to the given address
func ApiGet(r *http.Request, url string, rb interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", r.Header.Get("Authorization"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	json.NewDecoder(resp.Body).Decode(rb)
	resp.Body.Close()
	return nil
}

// http websocket upgrader
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// AuthToken is a PerRPCCredentials interface,
// as the augument of grpc.WithPerRPCCredentials()
type AuthToken string

// GetRequestMetadata is a required PerRPCCredentials interface method
// to inject token to request metadata
func (t AuthToken) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{"authorization": string(t)}, nil
}

// RequireTransportSecurity is a required PerRPCCredentials interface method
// to mandate use of TLS transport layer
func (t AuthToken) RequireTransportSecurity() bool {
	return true
}

// MongoOpr define methods for mongo database operation
type MongoOpr struct {
	Mdb     *mongo.Database
	Mcoll   *mongo.Collection
	Mctx    context.Context
	Mcancel context.CancelFunc
}

func (dba *MongoOpr) Set(col string) {
	dba.Mctx, dba.Mcancel = context.WithTimeout(context.Background(), 10*time.Second)
	dba.Mcoll = dba.Mdb.Collection(col)
}

// GetID find the exact data based on the given mongo _id and projection
// return false if no found
func (dba *MongoOpr) GetID(res interface{}, id primitive.ObjectID, projection map[string]interface{}) (bool, error) {
	f := bson.D{{"_id", id}}
	p, err := bson.Marshal(projection)
	if err != nil {
		return false, err
	}
	if err := dba.Mcoll.FindOne(dba.Mctx, f, options.FindOne().SetProjection(p)).Decode(res); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// GetData find the first data based on the given filter and projection
// return false if no found
func (dba *MongoOpr) GetData(res interface{}, filter, projection map[string]interface{}) (bool, error) {
	f, err := bson.Marshal(filter)
	if err != nil {
		return false, err
	}
	p, err := bson.Marshal(projection)
	if err != nil {
		return false, err
	}
	if err := dba.Mcoll.FindOne(dba.Mctx, f, options.FindOne().SetProjection(p)).Decode(res); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// GetDataset returns ordered data set based on given filter and projection
func (dba *MongoOpr) GetDataset(res interface{}, filter, projection, order map[string]interface{}) error {
	f, err := bson.Marshal(filter)
	if err != nil {
		return err
	}
	p, err := bson.Marshal(projection)
	if err != nil {
		return err
	}
	o, err := bson.Marshal(order)
	if err != nil {
		return err
	}
	// find all but only return projected fields
	cursor, err := dba.Mcoll.Find(dba.Mctx, f, options.Find().SetSort(o).SetProjection(p))
	if err != nil {
		return err
	}
	if err := cursor.All(dba.Mctx, res); err != nil {
		return err
	}
	return nil
}
