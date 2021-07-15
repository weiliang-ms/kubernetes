package main

import (
	"fmt"
	"reflect"
	"sort"
)

type Empty struct{}
type String map[string]Empty
type Fnc func(input string) string
type sortableSliceOfString []string

func (s sortableSliceOfString) Len() int           { return len(s) }
func (s sortableSliceOfString) Less(i, j int) bool { return lessString(s[i], s[j]) }
func (s sortableSliceOfString) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func main() {
	initController()
}

func initController() {
	mapA := map[string]Fnc{}
	mapA["etcd"] = etcdFnc
	mapA["job"] = jobFnc
	s := StringKeySet(mapA)
	s.Insert("namespace-controller")
	fmt.Println(s.List())
}

func etcdFnc(string) string {
	return "etcd"
}

func jobFnc(string) string {
	return "job-controller"
}

func StringKeySet(theMap interface{}) String {
	v := reflect.ValueOf(theMap)
	ret := String{}

	for _, keyValue := range v.MapKeys() {
		ret.Insert(keyValue.Interface().(string))
	}
	return ret
}

// Insert adds items to the set.
func (s String) Insert(items ...string) String {
	for _, item := range items {
		s[item] = Empty{}
	}
	return s
}

// List returns the contents as a sorted string slice.
func (s String) List() []string {
	res := make(sortableSliceOfString, 0, len(s))
	for key := range s {
		res = append(res, key)
	}
	sort.Sort(res)
	return []string(res)
}

func lessString(lhs, rhs string) bool {
	return lhs < rhs
}
