package util

import (
	"math"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"
)

func init() {
	// config package level default logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

/* ****************************************
map manipulating
**************************************** */

// KeySlice returns a slice of map keys
func KeySlice(m interface{}) []string {
	xm, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(xm))
	for k := range xm {
		keys = append(keys, k)
	}
	return keys
}

// JoinKeys returns a string of map keys concatnated by a given string
func JoinKeys(m interface{}, s string) string {
	return strings.Join(KeySlice(m), s)
}

// AssertMapSliceString resolves a map[string][]string interface to []string
func AssertMapSliceString(x interface{}, k string) []string {
	xm, ok := x.(map[string]interface{})
	if !ok {
		return nil
	}
	ms, ok := xm[k]
	if !ok {
		return nil
	}
	xs, ok := ms.([]interface{})
	if !ok {
		return nil
	}
	str := []string{}
	for _, st := range xs {
		sts, ok := st.(string)
		if !ok {
			return nil
		}
		str = append(str, sts)
	}
	return str
}

// AssertMapString resolves a map[string]string interface to string
func AssertMapString(x interface{}, k string) string {
	xm, ok := x.(map[string]interface{})
	if !ok {
		return ""
	}
	ms, ok := xm[k]
	if !ok {
		return ""
	}
	str, ok := ms.(string)
	if !ok {
		return ""
	}
	return str
}

/* ****************************************
map sorting functions
**************************************** */

// Compare encapsulates a string comparison function
type Compare func(str1, str2 string) bool

// NatureOrder creates a Compare instance operated on nature order of strings
func NatureOrder() Compare {
	retrieveNumber := func(str1, str2 string) bool {
		return extractNumberFromString(str1, 0) < extractNumberFromString(str2, 0)
	}
	return Compare(retrieveNumber)
}

// Sort the string list based on Compare func
func (cmp Compare) Sort(strs []string) {
	strSort := &strSorter{
		strs: strs,
		cmp:  cmp,
	}
	sort.Sort(strSort)
}

type strSorter struct {
	strs []string
	cmp  func(str1, str2 string) bool
}

func extractNumberFromString(str string, size int) (num int) {

	strSlice := make([]string, 0)
	for _, v := range str {
		if unicode.IsDigit(v) {
			strSlice = append(strSlice, string(v))
		}
	}

	if size == 0 { // default
		num, err := strconv.Atoi(strings.Join(strSlice, ""))
		if err != nil {
			return 0
		}
		return num
	}
	num, err := strconv.Atoi(strSlice[size-1])
	if err != nil {
		return 0
	}
	return num
}

func (s *strSorter) Len() int { return len(s.strs) }

func (s *strSorter) Swap(i, j int) { s.strs[i], s.strs[j] = s.strs[j], s.strs[i] }

func (s *strSorter) Less(i, j int) bool { return s.cmp(s.strs[i], s.strs[j]) }

// SortMapByField sorts a list of map by the value of a given key
// either on the provided order or natural ascend
// string with numbers or int/int64 can be sorted in their natural order
func SortMapByField(m []map[string]interface{}, f string, tseq []string) []map[string]interface{} {

	withKey := []map[string]interface{}{}
	withoutKey := []map[string]interface{}{}

	tseqm := make(map[string]struct{})
	for _, em := range m {
		v, ok := em[f]
		if !ok {
			withoutKey = append(withoutKey, em)
			continue
		}
		var gv string
		switch uv := v.(type) {
		case string:
			gv = uv
		case int:
			gv = strconv.Itoa(uv)
		case int64:
			gv = strconv.FormatInt(int64(uv), 10)
		default:
			withoutKey = append(withoutKey, em)
			continue
		}
		tseqm[gv] = struct{}{}
		withKey = append(withKey, em)
	}
	// sort by field f based on the natural ascend order
	if tseq == nil {
		tseq = []string{}
		for em := range tseqm {
			tseq = append(tseq, em)
		}
		// sort the value list
		//sort.Strings(tseq)
		NatureOrder().Sort(tseq)
	}

	// otherwise sort by field f based on the sequence of argument list
	sorted := []map[string]interface{}{}
	for _, k := range tseq {
		for i := 0; i < len(withKey); i++ {
			q := withKey[0]
			withKey = withKey[1:]
			var mv string
			switch uuv := q[f].(type) {
			case string:
				mv = uuv
			case int:
				mv = strconv.Itoa(uuv)
			case int64:
				mv = strconv.FormatInt(int64(uuv), 10)
			default:
			}
			if mv == k {
				sorted = append(sorted, q)
			} else {
				withKey = append(withKey, q)
			}
		}
	}
	withKey = append(withKey, withoutKey...)
	sorted = append(sorted, withKey...)
	return sorted
}

// SortMapByTwoFields sorts a list of map by the value of two given keys
// either on the provided order or natural ascend
// string with numbers or int/int64 can be sorted in their natural order
func SortMapByTwoFields(m []map[string]interface{}, f1 string, fseq []string, f2 string, sseq []string) []map[string]interface{} {

	withKey := []map[string]interface{}{}
	withoutKey := []map[string]interface{}{}

	tseqm := make(map[string]struct{})
	for _, em := range m {
		v, ok := em[f1]
		if !ok {
			withoutKey = append(withoutKey, em)
			continue
		}
		var gv string
		switch uv := v.(type) {
		case string:
			gv = uv
		case int:
			gv = strconv.Itoa(uv)
		case int64:
			gv = strconv.FormatInt(int64(uv), 10)
		default:
			withoutKey = append(withoutKey, em)
			continue
		}
		tseqm[gv] = struct{}{}
		withKey = append(withKey, em)
	}
	// sort by field f1 based on the natural ascend order
	if fseq == nil {
		fseq = []string{}
		for em := range tseqm {
			fseq = append(fseq, em)
		}
		// sort the value list
		//sort.Strings(fseq)
		NatureOrder().Sort(fseq)
	}

	// otherwise sort by field f1 based on the sequence of argument list
	sorted := []map[string]interface{}{}
	for _, k := range fseq {
		tempSorted := []map[string]interface{}{}
		for i := 0; i < len(withKey); i++ {
			q := withKey[0]
			withKey = withKey[1:]
			var mv string
			switch uuv := q[f1].(type) {
			case string:
				mv = uuv
			case int:
				mv = strconv.Itoa(uuv)
			case int64:
				mv = strconv.FormatInt(int64(uuv), 10)
			default:
			}
			if mv == k {
				tempSorted = append(tempSorted, q)
			} else {
				withKey = append(withKey, q)
			}
		}
		sorted = append(sorted, SortMapByField(tempSorted, f2, sseq)...)
	}
	withKey = append(SortMapByField(withKey, f2, sseq), SortMapByField(withoutKey, f2, sseq)...)
	sorted = append(sorted, withKey...)
	return sorted
}

/* ****************************************
string slice and map keys comparing functions
**************************************** */
// InStrings returns true if string in the slice of strings
func InStrings(e string, s []string) bool {
	for _, se := range s {
		if se == e {
			return true
		}
	}
	return false
}

// RemoveEmptyString remove the empty string from a slice
func RemoveEmptyString(s []string) (e []string) {
	for _, se := range s {
		if se != "" {
			e = append(e, se)
		}
	}
	return
}

// TrmEmptyString trim white spaces of all members before remove the empty elements from a slice
func TrmEmptyString(s []string) (e []string) {
	for _, se := range s {
		se = strings.TrimSpace(se)
		if se != "" {
			e = append(e, se)
		}
	}
	return
}

// Sckm returns true if a string slice is equal to the keys of a map
// regardless the order or repeat elements in the slice
func Sckm(s []string, m interface{}) bool {
	su := make(map[string]int, len(s))
	for _, k := range s {
		su[k]++
	}
	mv := reflect.ValueOf(m)
	if len(su) != mv.Len() {
		return false
	}
	for suk := range su {
		if !mv.MapIndex(reflect.ValueOf(suk)).IsValid() {
			return false
		}
	}
	return true
}

// Sccno returns true if two string slices are equal, regardless order and repeat
func Sccno(s1, s2 []string) bool {
	su1 := make(map[string]int)
	su2 := make(map[string]int)
	for _, k := range s1 {
		su1[k]++
	}
	for _, k := range s2 {
		su2[k]++
	}
	if len(su1) != len(su2) {
		return false
	}
	for suk := range su1 {
		if _, ok := su2[suk]; !ok {
			return false
		}
	}
	return true
}

/* This is not complete, only work on []string
func SliceCompareOrderless(s1, s2 interface{}) bool {
	sv1 := reflect.ValueOf(s1)
	sv2 := reflect.ValueOf(s2)
	if sv1.Len() != sv2.Len() { return false }	// faster to return

    	// create a map of string -> int
    	diff := make(map[string]int, sv2.Len())
    	for i:=0; i<sv2.Len(); i++ {
       		diff[sv2.Index(i).String()]++		// String() always converts
    	}
    	for i:=0; i<sv1.Len(); i++ {
		n := sv1.Index(i).String()
        	if _, ok := diff[n]; !ok {
           		return false
        	}
        	diff[n] -= 1
        	if diff[n] == 0 {
            		delete(diff, n)
        	}
    	}
    	if len(diff) == 0 { return true }
    	return false
}
*/

/* ****************************************
random string functions
**************************************** */
// charset is the numeric and alphabetic character set
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// StringWithCharset generates random string on a given length and character set
func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandString generates random numeric and alphabetic string on a given length
func RandString(length int) string {
	return StringWithCharset(length, charset)
}

/* ****************************************
ip address functions
**************************************** */

// IP holds IPv4 and IPv6 data structure and provides operations on it
type IP struct {
	V6   bool // IPv4 - false, IPv6 - true
	Addr string
	Mask int
}

// StringToIP converts x.x.x.x/24 or f8ae:12::1/128 to IP obj, default mask is 32 or 128
func StringToIP(s string) *IP {
	var ip IP
	var err error
	if strings.Contains(s, ":") {
		ip.V6 = true
	} else if !strings.Contains(s, ".") {
		return nil
	}
	sst := strings.Split(s, "/")
	switch len(sst) {
	case 1:
		ip.Addr = sst[0]
		if ip.V6 {
			ip.Mask = 128
		} else {
			ip.Mask = 32
		}
	case 2:
		ip.Addr = sst[0]
		ip.Mask, err = strconv.Atoi(sst[1])
		if err != nil {
			return nil
		}
		// more strict check add here
	default:
		return nil
	}
	return &ip
}

// ListToIps converts a slice of IP address string to a IP obj slice
func ListToIps(l []string) (i []*IP) {
	for _, ip := range l {
		i = append(i, StringToIP(ip))
	}
	return
}

// String converts IP to a string like x.x.x.x/32
func (ip *IP) String() string {
	return ip.Addr + "/" + strconv.Itoa(ip.Mask)
}

// SameIP returns true if two IP have the same address and mask
func (ip *IP) SameIP(t *IP) bool {
	if ip.Addr == t.Addr && ip.Mask == t.Mask {
		return true
	}
	return false
}

/* ****************************************
timestamp functions
**************************************** */

// StringToEpoch converts string to UTC epoch seconds
func StringToEpoch(s string) (int64, error) {
	formats := []string{
		"2006-01-02 15:04:05 MST", // JUNOS
		time.UnixDate,             // SROS, Ubuntu
	}
	TzInfo := map[string]int64{
		"UTC":  0,
		"GMT":  0,
		"AST":  -14400,
		"EST":  -18000,
		"EDT":  -14400,
		"CST":  -21600,
		"CDT":  -18000,
		"MST":  -25200,
		"MDT":  -21600,
		"PST":  -28800,
		"PDT":  -25200,
		"AKST": -32400,
		"AKDT": -28800,
		"HST":  -36000,
		"HAST": -36000,
		"HADT": -32400,
		"SST":  -39600,
		"SDT":  -36000,
		"CHST": 36000,
	}
	var err error
	for _, format := range formats {
		t2, err := time.Parse(format, s)
		if err == nil {
			zone := t2.Location().String()
			return t2.Unix() - TzInfo[zone], nil
		}
	}
	return 0, err
}

// EpochToString converts a int64 UTC epoch to a string
func EpochToString(t int64) string {
	return time.Unix(t, 0).Format(time.UnixDate)
}

/* ****************************************
utility functions
**************************************** */

// UpDown converts bool values to Up/Down string
func UpDown(adm, opr bool) (s string) {
	if adm {
		s = "Up/"
	} else {
		s = "Down/"
	}
	if opr {
		s += "Up"
	} else {
		s += "Down"
	}
	return
}

// LogWithFields attaches a slice of [k1,v1,k2,v2,...] to log entry
func LogWithFields(log *log.Entry, f []string) *log.Entry {
	for i := 0; i < len(f)-1; i += 2 {
		log = log.WithField(f[i], f[i+1])
	}
	return log
}

// RoundTo rounds a float to a given position, also a float type
func RoundTo(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}
