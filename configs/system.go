/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package configs

import "time"

// type and version
const Version = "cess-scheduler v0.5.2"

const (
	// Name is the name of the program
	Name = "cess-scheduler"
	// Description is the description of the program
	Description = "Implementation of Scheduling Service for Consensus Nodes"
	// NameSpace is the cached namespace
	NameSpace = "scheduler"
)

const (
	// BaseDir is the base directory where data is stored
	BaseDir = NameSpace
	// Data directory
	LogDir    = "log"
	CacheDir  = "cache"
	FileDir   = "file"
	TagDir    = "tag"
	FillerDir = "filler"
)

const (
	// BlockInterval is the time interval for generating blocks, in seconds
	BlockInterval = time.Second * time.Duration(6)
)
