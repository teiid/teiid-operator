/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// EventType --
type EventType string

const (
	// VdbDeleted --
	VdbDeleted EventType = "VDB Deleted"
)

// EventListener --
type EventListener interface {
	onEvent(eventType EventType, resource types.NamespacedName, payload interface{})
}

// EventSubscribers --
type EventSubscribers struct {
	handlers []EventListener
	mutex    sync.RWMutex
}

// Register adds an event handler for this event
func (u *EventSubscribers) Register(handler EventListener) {
	u.mutex.Lock()
	u.handlers = append(u.handlers, handler)
	u.mutex.Unlock()
}

// Trigger sends out an event with the payload
func (u *EventSubscribers) Trigger(eventType EventType, resource types.NamespacedName, payload interface{}) {
	u.mutex.Lock()
	for _, handler := range u.handlers {
		handler.onEvent(eventType, resource, payload)
	}
	u.mutex.Unlock()
}
