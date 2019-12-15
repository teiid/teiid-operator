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

// PrometheusConfig --
func PrometheusConfig() string {
	return `
    startDelaySecs: 5
    ssl: false
    blacklistObjectNames: ["java.lang:*"]
    rules:
    # Runtime/Engine level
      - pattern: 'org.teiid<type=Runtime><>TotalRequestsProcessed'
        name: org.teiid.TotalRequestsProcessed
        help: Total Requests Processed
        type: COUNTER
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><>WaitingRequestsCount'
        name: org.teiid.WaitingRequestsCount
        help: Requests that are waiting to begin processing
        type: GAUGE
        labels:
            type: runtime      
      - pattern: 'org.teiid<type=Runtime><>ActiveEngineThreadCount'
        name: org.teiid.ActiveEngineThreadCount
        help: Number of Engine Threads Currently Working
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><>QueuedEngineWorkItems'
        name: org.teiid.QueuedEngineWorkItems
        help: Number of Queued Work Items
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><>LongRunningRequestCount'
        name: org.teiid.LongRunningRequestCount
        help: Number of Long Running Requests
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><>TotalOutOfDiskErrors'
        name: org.teiid.TotalOutOfDiskErrors
        help: Total Buffer Out Of Disk Errors
        type: COUNTER
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><>PercentBufferDiskSpaceInUse'
        name: org.teiid.PercentBufferDiskSpaceInUse
        help: Percent Buffer Disk Space In Use
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><EngineStatisticsBean>sessionCount'
        name: org.teiid.SessionCount
        help: Number of Client Sessions
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><EngineStatisticsBean>diskSpaceUsedInMB'
        name: org.teiid.DiskSpaceUsedInMB
        help: Amount of Disk MB in Use
        type: GAUGE
        labels:
            type: runtime
      - pattern: 'org.teiid<type=Runtime><EngineStatisticsBean>activePlanCount'
        name: org.teiid.ActiveRequestCount
        help: Number of Actively Processing Requests
        type: GAUGE
        labels:
            type: runtime
            
    #cache
      - pattern: 'org.teiid<type=Cache, name=(\w*)><>RequestCount'
        name: org.teiid.CacheRequestCount
        help: Number of Cache Reads
        type: GAUGE
        labels:
            type: cache
            entry: $1
      - pattern: 'org.teiid<type=Cache, name=(\w*)><>TotalEntries'
        name: org.teiid.CacheTotalEntries
        help: Number of Cache Entries
        type: GAUGE
        labels:
            type: cache
            entry: $1
      - pattern: 'org.teiid<type=Cache, name=(\w*)><>HitRatio'
        name: org.teiid.CacheHitRatio
        help: Hits / Total Attempts
        type: GAUGE
        labels:
            type: cache
            entry: $1

    # Spring?

    # Services - CXF from s2i default config
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>NumLogicalRuntimeFaults'
        name: org.apache.cxf.NumLogicalRuntimeFaults
        help: Number of logical runtime faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>NumLogicalRuntimeFaults'
        name: org.apache.cxf.NumLogicalRuntimeFaults
        help: Number of logical runtime faults
        type: GAUGE
        labels:
          bus.id: $1
          type: $2
          service: $3
          port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>AvgResponseTime'
        name: org.apache.cxf.AvgResponseTime
        help: Average Response Time
        type: GAUGE
        labels:
          bus.id: $1
          type: $2
          service: $3
          port: $4
          operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>AvgResponseTime'
        name: org.apache.cxf.AvgResponseTime
        help: Average Response Time
        type: GAUGE
        labels:
              bus.id: $1
              type: $2
              service: $3
              port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>NumInvocations'
        name: org.apache.cxf.NumInvocations
        help: Number of invocations
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>NumInvocations'
        name: org.apache.cxf.NumInvocations
        help: Number of invocations
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>MaxResponseTime'
        name: org.apache.cxf.MaxResponseTime
        help: Maximum Response Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>MaxResponseTime'
        name: org.apache.cxf.MaxResponseTime
        help: Maximum Response Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>MinResponseTime'
        name: org.apache.cxf.MinResponseTime
        help: Minimum Response Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>MinResponseTime'
        name: org.apache.cxf.MinResponseTime
        help: Minimum Response Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>TotalHandlingTime'
        name: org.apache.cxf.TotalHandlingTime
        help: Total Handling Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>TotalHandlingTime'
        name: org.apache.cxf.TotalHandlingTime
        help: Total Handling Time
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>NumRuntimeFaults'
        name: org.apache.cxf.NumRuntimeFaults
        help: Number of runtime faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>NumRuntimeFaults'
        name: org.apache.cxf.NumRuntimeFaults
        help: Number of runtime faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>NumUnCheckedApplicationFaults'
        name: org.apache.cxf.NumUnCheckedApplicationFaults
        help: Number of unchecked application faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>NumUnCheckedApplicationFaults'
        name: org.apache.cxf.NumUnCheckedApplicationFaults
        help: Number of unchecked application faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+), operation=([^,]+)><>NumCheckedApplicationFaults'
        name: org.apache.cxf.NumCheckedApplicationFaults
        help: Number of checked application faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
            operation: $5
      - pattern: 'org.apache.cxf<bus.id=([^,]+), type=([^,]+), service=([^,]+), port=([^,]+)><>NumCheckedApplicationFaults'
        name: org.apache.cxf.NumCheckedApplicationFaults
        help: Number of checked application faults
        type: GAUGE
        labels:
            bus.id: $1
            type: $2
            service: $3
            port: $4
    `
}
