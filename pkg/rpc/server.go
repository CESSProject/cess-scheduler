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

package rpc

import (
	"sync"

	mapset "github.com/deckarep/golang-set"
)

type Server struct {
	running int32
	conns   mapset.Set

	handleWg sync.WaitGroup
	router   *serviceRouter
}

func NewServer() *Server {
	s := &Server{
		conns:  mapset.NewSet(),
		router: newServiceRouter(),
	}
	return s
}

func (s *Server) serve(codec *websocketCodec) {
	c := &SrvConn{
		srv:   s,
		codec: codec,
	}
	// Add the conn to the set so it can be closed by Stop.
	s.conns.Add(c)
	defer s.conns.Remove(c)

	go c.readLoop()
}

func (s *Server) Register(name string, service interface{}) error {
	return s.router.registerName(name, service)
}

func (s *Server) Close() {
	s.conns.Each(func(c interface{}) bool {
		c.(websocketCodec).close()
		return true
	})
	s.handleWg.Wait()
}
