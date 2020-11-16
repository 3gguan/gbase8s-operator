package controllers

import (
	"fmt"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type PodStatusChangedPredicate struct {
	predicate.Funcs
}

func (r *PodStatusChangedPredicate) Update(e event.UpdateEvent) bool {
	a := reflect.TypeOf(e.MetaOld)
	fmt.Println(a)
	return true
}
