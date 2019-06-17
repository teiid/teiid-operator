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

package virtualdatabase

import (
	"context"

	"github.com/teiid/teiid-operator/pkg/apis/vdb/v1alpha1"
	teiidclient "github.com/teiid/teiid-operator/pkg/client"
	logt "github.com/teiid/teiid-operator/pkg/util/log"
)

// Action --
type Action interface {
	teiidclient.Injectable

	// a user friendly name for the action
	Name() string

	// returns true if the action can handle the vdb
	CanHandle(vdb *v1alpha1.VirtualDatabase) bool

	// executes the handling function
	Handle(ctx context.Context, vdb *v1alpha1.VirtualDatabase) error

	// Inject virtualization logger
	InjectLogger(logt.Logger)
}

type baseAction struct {
	client teiidclient.Client
	Log    logt.Logger
}

func (action *baseAction) InjectClient(client teiidclient.Client) {
	action.client = client
}

func (action *baseAction) InjectLogger(log logt.Logger) {
	action.Log = log
}
