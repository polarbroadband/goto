package util

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/kr/pretty"

	log "github.com/sirupsen/logrus"
)

func init() {
	// config package level default logger
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
}

/* ****************************************
concurrent map operation
**************************************** */

type DynaStore struct {
	Pool map[string]interface{}
	lock *sync.RWMutex
}

func NewDynaStore(c ...map[string]interface{}) *DynaStore {
	if len(c) < 1 {
		return &DynaStore{map[string]interface{}{}, &sync.RWMutex{}}
	}
	pool := DynaStore{c[0], &sync.RWMutex{}}
	for _, cc := range c[1:] {
		pool.Update(cc)
	}
	return &pool
}

// Len retrieve the current size of pool
func (s *DynaStore) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.Pool)
}

// Exist return true if key exists in pool
func (s *DynaStore) Exist(k string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, exist := s.Pool[k]
	return exist
}

// Keys return key list of the pool
func (s *DynaStore) Keys() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	keys := []string{}
	for k, _ := range s.Pool {
		keys = append(keys, k)
	}
	return keys
}

// Update add key/value pairs to the pool, overwrite if key duplicated
func (s *DynaStore) Update(d map[string]interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for k, v := range d {
		s.Pool[k] = v
	}
}

// Get retrieve value of given key as interface{}
func (s *DynaStore) Get(k string) interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Pool[k]
}

// Fetch retrieve value of given key as interface{}
func (s *DynaStore) Fetch(k string) interface{} {
	s.lock.RLock()
	r := s.Pool[k]
	s.lock.RUnlock()
	s.lock.Lock()
	delete(s.Pool, k)
	s.lock.Unlock()
	return r
}

// GetString retrieve string value, return "" if invalid
func (s *DynaStore) GetString(k string) string {
	if res, ok := s.Get(k).(string); ok {
		return strings.TrimSpace(res)
	}
	return ""
}

// GetBool retrieve bool value, return false if invalid
func (s *DynaStore) GetBool(k string) bool {
	if s, ok := s.Get(k).(bool); ok {
		return s
	}
	return false
}

// GetStringArr retrieve a string slice, return empty if invalid
func (s *DynaStore) GetStringArr(k string) []string {
	return TrmEmptyString(s.Get(k))
}

// GetMap retrieve embedded map, return nil if invalid
func (s *DynaStore) GetMap(k string) map[string]interface{} {
	if res, ok := s.Get(k).(map[string]interface{}); ok {
		return res
	}
	return nil
}

// GetInt64 retrieve number value as int64, return 0 if invalid
// convert int, float64 to int64
// convert string i.e "98" or "9.12" to int64
func (s *DynaStore) GetInt64(k string) int64 {
	if m, err := strconv.ParseFloat(fmt.Sprintf("%v", s.Get(k)), 64); err == nil {
		return int64(math.Round(m))
	}
	return 0
}

// GetFloat retrieve number value as float64, return 0 if invalid
func (s *DynaStore) GetFloat(k string) float64 {
	if m, err := strconv.ParseFloat(fmt.Sprintf("%v", s.Get(k)), 64); err == nil {
		return m
	}
	return 0
}

/* ****************************************
map manipulating
**************************************** */

// MapMerge merge two map[string]interface{}
func MapMerge(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range b {
		a[k] = v
	}
	return a
}

// TrimMap removes leading and tailing white spaces from all members
func TrimMap(m map[string]string) map[string]string {
	n := make(map[string]string)
	for k, v := range m {
		n[k] = strings.TrimSpace(v)
	}
	return n
}

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

// DigValue walk through the embedded map[string]interface{} base on a sequence of keys
// retrieve the value of last key, return nil and broken branch
func DigValue(m interface{}, keys ...string) (interface{}, string) {
	get := func(m interface{}, key string) interface{} {
		if mm, ok := m.(map[string]interface{}); ok {
			return mm[key]
		}
		return nil
	}

	if _, ok := m.(map[string]interface{}); !ok {
		return nil, "/"
	}

	broken := ""
	v := m
	for _, key := range keys {
		v = get(v, key)
		if v == nil {
			broken += "/" + key
			break
		}
	}
	return v, broken
}

// DigValue walk through the embedded map[string]interface{} base on a sequence of keys
// retrieve the string value of last key, return nil and broken branch
func DigString(m interface{}, keys ...string) (string, string) {
	r, broken := DigValue(m, keys...)
	if r != nil {
		if v, ok := r.(string); ok {
			return v, ""
		}
		return "", fmt.Sprintf("invalid type %T, value %v", r, r)
	}
	return "", broken
}

// DigValue walk through the embedded map[string]interface{} base on a sequence of keys
// retrieve the float64 value of last key
// will attemp to convert string i.e "1.43" to float64 if possible
// return nil and broken branch
func DigFloat(m interface{}, keys ...string) (float64, string) {
	r, broken := DigValue(m, keys...)
	if r != nil {
		if v, ok := r.(float64); ok {
			return v, ""
		} else if v, ok := r.(string); ok {
			if sv, err := strconv.ParseFloat(v, 64); err == nil {
				return sv, ""
			}
		}
		return 0, fmt.Sprintf("invalid type %T, value %v", r, r)
	}
	return 0, broken
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
// ConvToStrings converts a interface{} to []string
// underlying type []string or []interface{} only
// logging and return empty []string for any invalid input
func ConvToStrings(s interface{}) []string {
	process := func(t []interface{}) (oprS []string) {
		for _, e := range t {
			if es, ok := e.(string); ok {
				oprS = append(oprS, es)
			} else {
				log.Warn("ConvToStrings returns empty: at least one member of the given slice is not a string")
				return []string{}
			}
		}
		return
	}
	switch ts := s.(type) {
	case []string:
		return ts
	case *[]string:
		return *ts
	case []interface{}:
		return process(ts)
	case *[]interface{}:
		return process(*ts)
	default:
		log.Warn("ConvToStrings returns empty: neither []string nor []interface{}")
		return []string{}
	}
}

// InStrings returns true if string in the slice of strings
func InStrings(e string, s interface{}) bool {
	for _, se := range ConvToStrings(s) {
		if se == e {
			return true
		}
	}
	return false
}

// RemoveEmptyString remove the empty string from a slice
func RemoveEmptyString(s interface{}) (e []string) {
	for _, se := range ConvToStrings(s) {
		if se != "" {
			e = append(e, se)
		}
	}
	return
}

// TrmEmptyString trim white spaces of all members before remove the empty elements from a slice
func TrmEmptyString(s interface{}) (e []string) {
	for _, se := range ConvToStrings(s) {
		se = strings.TrimSpace(se)
		if se != "" {
			e = append(e, se)
		}
	}
	return
}

// TrmStrings trim white spaces of all members but keep the empty elements
func TrmStrings(s interface{}) (e []string) {
	for _, se := range ConvToStrings(s) {
		se = strings.TrimSpace(se)
		e = append(e, se)
	}
	return
}

// RevStringsOrder revers the order of string slice
func RevStringsOrder(s interface{}) (e []string) {
	ss := ConvToStrings(s)
	for i := len(ss) - 1; i >= 0; i-- {
		e = append(e, ss[i])
	}
	return
}

// IndexStrings returns index of element in given reference of string slice
// return -1 if not found
func IndexStrings(s interface{}, k string) int {
	ss := ConvToStrings(s)
	for p, v := range ss {
		if v == k {
			return p
		}
	}
	return -1
}

// Truncate a string to given length
func Truncate(s string, maxLength int) string {
	if len(s) > maxLength+1 {
		s = s[0:maxLength] + "..."
	}
	return s
}

// StrInterpolate interpolate and extand a symbol string to a string list
// the word to be calaulate mark as "^0-4$" to 0,1,2,3,4
// the word to be calaulate mark as "^0-5+2$" to 0,2,4
// the word to be calaulate mark as "^34, er_8, 9 8y$" to 34,er_8,9 8y
/* "I had ^2 -3$ eggs for ^breakfast, dinner$" to be change to
I had 2 eggs for breakfast
I had 2 eggs for dinner
I had 3 eggs for breakfast
I had 3 eggs for dinner
*/
func StrInterpolate(s string) *[]string {
	r := []string{s}
	re := regexp.MustCompile(`(?:\^\s*(\d+)\s*-\s*(\d+)\s*(?:\+(\d+))?\$)|(?:\^[\w\s,]+\$)`)
	fd := re.FindAllStringSubmatch(s, -1)
	if len(fd) < 1 {
		return nil
	}
	for _, elem := range fd {
		ks := []string{}
		if qt := regexp.MustCompile(`\^([\w\s,]+)\$`).FindStringSubmatch(elem[0]); len(qt) > 1 {
			for _, qts := range strings.Split(qt[1], ",") {
				ks = append(ks, strings.TrimSpace(qts))
			}
		} else {
			start, err := strconv.ParseInt(elem[1], 10, 64)
			if err != nil {
				return nil
			}
			end, err := strconv.ParseInt(elem[2], 10, 64)
			if err != nil {
				return nil
			}
			ks = append(ks, elem[1])
			var step int64 = 1
			if elem[3] != "" {
				step, err = strconv.ParseInt(elem[3], 10, 64)
				if err != nil {
					return nil
				}
			}
			for {
				start += step
				if start > end {
					break
				}
				ks = append(ks, strconv.FormatInt(start, 10))
			}
		}
		tr := []string{}
		for _, ri := range r {
			for _, inpt := range ks {
				tr = append(tr, strings.Replace(ri, elem[0], inpt, 1))
			}
		}
		r = tr
	}
	return &r
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

// InSlice check if a given value is in a slice
// use reflect DeepEqual as comparison method
func InSlice(v interface{}, s []interface{}) bool {
	for _, m := range s {
		if reflect.DeepEqual(v, m) {
			return true
		}
	}
	return false
}

// GetValueIgnoreCase get value of key regardless case
// return nil if not found
func GetValueIgnoreCase(m map[string]interface{}, key string) interface{} {
	for k, v := range m {
		if strings.ToLower(k) == strings.ToLower(key) {
			return v
		}
	}
	return nil
}

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

// StringToDuration converts a duration string (8y10w7d6h5m20s)to time.Duration
// add year, week and day unit support on top of time.ParseDuration
// return 0 if invalid string
func StringToDuration(s string) time.Duration {
	ss := regexp.MustCompile(`^(?:(\d+)y)?(?:(\d+)w)?(?:(\d+)d)?([\dhms]+)?$`).FindStringSubmatch(strings.ToLower(s))
	if len(ss) == 0 {
		return time.Duration(0)
	}
	dur := time.Duration(0)
	if ss[1] != "" { // year
		if num, e := strconv.ParseInt(ss[1], 10, 64); e != nil {
			return time.Duration(0)
		} else {
			dur += time.Duration(num * 365 * 24 * 3600 * 1000000000)
		}
	}
	if ss[2] != "" { // week
		if num, e := strconv.ParseInt(ss[2], 10, 64); e != nil {
			return time.Duration(0)
		} else {
			dur += time.Duration(num * 7 * 24 * 3600 * 1000000000)
		}
	}
	if ss[3] != "" { // day
		if num, e := strconv.ParseInt(ss[3], 10, 64); e != nil {
			return time.Duration(0)
		} else {
			dur += time.Duration(num * 24 * 3600 * 1000000000)
		}
	}
	st, _ := time.ParseDuration(ss[4]) // h:m:s
	return dur + st
}

// HMSToDuration converts 6:10:30 format string to time.Duration
func HMSToDuration(s string) time.Duration {
	temp := []string{"s", "m", "h"}
	ss := strings.Split(s, ":")
	if len(ss) > 3 || len(ss) < 1 {
		return time.Duration(0)
	}
	k := 0
	for i := len(ss) - 1; i >= 0; i-- {
		ss[i] = strings.TrimSpace(ss[i]) + temp[k]
		k += 1
	}
	p := strings.Join(ss, "")
	fmt.Println(p)
	r, _ := time.ParseDuration(p)
	return r
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

// GetEnvHashFrFile getting a k/v map of env var from a file in shell format
func GetEnvHashFrFile(fileName string) map[string]string {
	res := make(map[string]string)
	if data, err := ioutil.ReadFile(fileName); err == nil {
		re := regexp.MustCompile(`^([\w\.-]+)=([\w\.-]+)$`)
		for _, ln := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			m := re.FindStringSubmatch(strings.TrimSpace(ln))
			if len(m) == 0 {
				continue
			}
			if m[1] == "" {
				continue
			}
			res[m[1]] = m[2]
		}
	}
	return res
}

// GetEnvArrayFrFile getting an array of env var objects with "key" and "val" fields
// original sequence will be preserved
func GetEnvArrayFrFile(fileName string) []map[string]string {
	res := []map[string]string{}
	if data, err := ioutil.ReadFile(fileName); err == nil {
		fmt.Println(string(data))
		re := regexp.MustCompile(`^([\w\.-]+)=([\w\.-]+)$`)
		for _, ln := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			m := re.FindStringSubmatch(strings.TrimSpace(ln))
			if len(m) == 0 {
				continue
			}
			if m[1] == "" {
				continue
			}
			fmt.Println(m)
			res = append(res, map[string]string{"key": m[1], "val": m[2]})
		}
	}
	return res
}

// FileExist check if the File exist and produced the same MD5 checksum
func FileExist(path, chksum string) (error, bool, string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err, false, ""
	}
	fileSum := fmt.Sprintf("%x", md5.Sum(content))
	if chksum != "" && fileSum != chksum {
		return fmt.Errorf("file exist but failed MD5 check"), true, ""
	}
	return nil, true, fileSum
}

// Debug pretty prints the any go object
func Debug(note string, s interface{}) {
	pretty.Printf("\n\n--- %s ---\n%# v\n", note, s)
}

/* ****************************************
Error handling
**************************************** */

// function execution failure
type ExeErr string

func NewExeErr(f string, i ...string) ExeErr {
	r := fmt.Sprintf("func %s failed", f)
	if len(i) > 0 {
		r = strings.Join(i, "/") + " " + r
	}
	return ExeErr(r)
}
func (e ExeErr) String(err ...interface{}) string {
	if len(err) == 0 {
		return fmt.Sprintf("%v", e)
	}
	if len(err) == 1 {
		return fmt.Sprintf("%v, %v", e, err[0])
	}
	addErr := ""
	for _, er := range err[1:] {
		addErr += fmt.Sprintf(" %v", er)
	}
	return fmt.Sprintf("%v, %v:%s", e, err[0], addErr)
}
func (e ExeErr) Error(err ...interface{}) error {
	if len(err) == 0 {
		return fmt.Errorf("%v", e)
	}
	if len(err) == 1 {
		return fmt.Errorf("%v, %v", e, err[0])
	}
	addErr := ""
	for _, er := range err[1:] {
		addErr += fmt.Sprintf(" %v", er)
	}
	return fmt.Errorf("%v, %v:%s", e, err[0], addErr)
}
