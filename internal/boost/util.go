// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package boost

type namespacedObjects[T any] struct {
	objects map[string]map[string]T
}

func newNamespacedObjects[T any]() *namespacedObjects[T] {
	return &namespacedObjects[T]{
		objects: make(map[string]map[string]T),
	}
}

func (o *namespacedObjects[T]) Put(name, namespace string, object T) {
	namespaceObjects, ok := o.objects[namespace]
	if !ok {
		namespaceObjects = make(map[string]T)
		o.objects[namespace] = namespaceObjects
	}
	namespaceObjects[name] = object
}

func (o *namespacedObjects[T]) Get(name, namespace string) (T, bool) {
	namespaceObjects, ok := o.objects[namespace]
	if !ok {
		return *new(T), false
	}
	object, ok := namespaceObjects[name]
	return object, ok
}

func (o *namespacedObjects[T]) List(namespace string) []T {
	namespacedObjects, ok := o.objects[namespace]
	if !ok {
		return []T{}
	}
	result := make([]T, 0, len(namespacedObjects))
	for _, object := range namespacedObjects {
		result = append(result, object)
	}
	return result
}

func (o *namespacedObjects[T]) ListAll() []T {
	result := make([]T, 0)
	for _, namespaceObjects := range o.objects {
		for _, object := range namespaceObjects {
			result = append(result, object)
		}
	}
	return result
}

func (o *namespacedObjects[T]) Delete(name, namespace string) {
	namespaceObjects, ok := o.objects[namespace]
	if !ok {
		return
	}
	delete(namespaceObjects, name)
}
