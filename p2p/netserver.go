// Copyright 2017~2022 The Bottos Authors
// This file is part of the Bottos Chain library.
// Created by Rocket Core Team of Bottos.

// This program is free software: you can distribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Bottos.  If not, see <http://www.gnu.org/licenses/>.

// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package exec provides functions for executing WebAssembly bytecode.

/*
 * file description: the interface for WASM execution
 * @Author: Stewart Li
 * @Date:   2018-02-08
 * @Last Modified by:
 * @Last Modified time:
 */

package p2pserver

import (
	//"io"
	"fmt"
	"net"
	"sync"
	"time"
	"errors"
	"strings"
	//"reflect"
	"unsafe"
	//"crypto/sha1"
	"hash/fnv"
	"encoding/json"
	//"github.com/bottos-project/core/contract/msgpack"
)

type NetServer struct {
	config          *P2PConfig
	port            uint16
	addr            string

	listener        net.Listener

	seed_peer       []string

	neighborList    []*net.UDPAddr
	serverAddr      *net.UDPAddr
	socket          *net.UDPConn

	peerMap         map[uint64]*Peer

	publicKey       string           //todo

	time_interval   *time.Timer

	netLock         sync.RWMutex
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func NewNetServer(config *P2PConfig) *NetServer {
	if config == nil {
		fmt.Println("*ERROR* Parmeter is empty !!!")
		return nil
	}

	return &NetServer{
		config:        config,
		addr:          config.ServAddr,
		port:          config.ServPort,
		peerMap:       make(map[uint64]*Peer),
		time_interval: time.NewTimer(TIME_INTERVAL * time.Second),
	}
}

//start listener
func (serv *NetServer) Start() error {
	fmt.Println("netServer::Start()")

	go serv.Listening()

	return nil
}

//run accept
func (serv *NetServer) Listening() {
	fmt.Println("NetServer::Listening()")
	//listener, err := net.Listen("tcp", serv.addr+":"+fmt.Sprint(serv.port))
	listener, err := net.Listen("tcp", ":"+fmt.Sprint(serv.port))
	if err != nil {
		fmt.Println("*ERROR* Failed to listen at port: "+fmt.Sprint(serv.port))
		return
	}

	defer listener.Close()



	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("NetServer::Listening() Failed to accept")
			continue
		}

		fmt.Println("NetServer::Listening() conn = ",conn)

		go serv.HandleMsg(conn)
	}

}

//run accept
func (serv *NetServer) HandleMsg(conn net.Conn) {
	//defer conn.Close()
	//fmt.Println("netServer::HandleMsg()")
	data := make([]byte, 4096)
	var msg message

	len , err := conn.Read(data)
	if err != nil {
		fmt.Println("*WRAN* Can't read data from remote peer !!!")
		return
	}

	err = json.Unmarshal(data[0:len] , &msg)
	if err != nil {
		fmt.Println("*WRAN* Can't unmarshal data from remote peer !!!")
		return
	}
	fmt.Println("NetServer::Listening() Loop msg = ",msg)

	switch msg.MsgType {
	case request:
		//todo receive a connection request from other peer passively
		rsp := message {
			Src:      serv.addr,
			Dst:      msg.Src,
			MsgType:  response,
		}

		data , err := json.Marshal(rsp)
		if err != nil{
			fmt.Println("*WRAN* Failed to package the response message : ", err)
		}

		//fmt.Println("netServer::HandleMsg() request len(data) = ",len(data) , ", rsp = ",rsp," , conn.RemoteAddr = ",conn.RemoteAddr()," , conn.LocalAddr = ",conn.LocalAddr())

		//create a new conn to response the remote peer
		newconn , err := net.Dial("tcp", msg.Src+":"+fmt.Sprint(serv.port))
		if err != nil {
			fmt.Println("*ERROR* Failed to create a connection for remote server !!! err: ",err)
			return
		}

		len , err := newconn.Write(data)
		if err != nil {
			fmt.Println("*ERROR* Failed to send data to the remote server addr !!! err: ",err)
			return
		}

		if len < 0 {
			fmt.Println("*ERROR* Failed to send data to the remote server addr !!! err: ",err)
			return
		}

		//package remote peer info as "peer" struct and add it into peer list
		addr_port := msg.Src + ":" + fmt.Sprint(serv.port)
		peer := NewPeer(msg.Src , &newconn)
		peer_identify := Hash(addr_port)
		if _, ok := serv.peerMap[uint64(peer_identify)]; !ok {
			serv.peerMap[uint64(peer_identify)] = peer
		}

		fmt.Println("NetServer::HandleMsg() request from = ", msg.Src)

		//fmt.Println("<-------------------------------------------------------------->")
		/*
		for key , peer := range serv.peerMap {
			fmt.Println("NetServer::HandleMsg() request key = ",key," , peer = ",peer)
		}
		*/
		//fmt.Println("<-------------------------------------------------------------->")

	case response: //a response from my proactive connect
		//todo if the remote peer hadn't existed at local , add it into local
		fmt.Println("NetServer::HandleMsg() response to = ", msg.Src)
		if serv.IsExist(msg.Src , false) {
			//fmt.Println("the Peer had existed ! peer = ",peer)
			return
		}

		newconn , err := net.Dial("tcp", msg.Src+":"+fmt.Sprint(serv.port))
		if err != nil {
			fmt.Println("*ERROR* Failed to create a connection for remote server !!! err: ",err)
			return
		}

		addr_port := msg.Src + ":" + fmt.Sprint(serv.port)
		peer := NewPeer(msg.Src , &newconn)
		peer_identify := Hash(addr_port)
		if _, ok := serv.peerMap[uint64(peer_identify)]; !ok {
			serv.peerMap[uint64(peer_identify)] = peer
		}


		//fmt.Println("<-------------------------------------------------------------->")
		//fmt.Println("NetServer::HandleMsg() response to = ", msg.Src)
		/*
		for key , peer := range serv.peerMap {
			fmt.Println("NetServer::HandleMsg() response key = ",key," , peer = ",peer)
		}
		*/
		//fmt.Println("<-------------------------------------------------------------->")
		//fmt.Println("NetServer::HandleMsg() response - netServer::Listening() len(serv.peerMap) = ",len(serv.peerMap))
	}

	return
}

func (serv *NetServer) ActiveSeeds() error {
	fmt.Println("p2pServer::ActiveSeeds()")
	for {
		select {
		case <- serv.time_interval.C:
			serv.ConnectSeeds()
			serv.WatchStatus()
			serv.ResetTimer()
		}
	}
}

//reset time to start timer for a new round
func  (serv *NetServer) ResetTimer ()  {
	serv.time_interval.Stop()
	serv.time_interval.Reset(time.Second * TIME_INTERVAL)
}

//connect seed during start p2p server
func (serv *NetServer) ConnectSeeds() error {

	//fmt.Println("p2pServer::ConnectSeeds()")

	/*
	for key, peer := range serv.peerMap {
		fmt.Println("NetServer::ConnectSeeds() key = ", key, " , peer = ", peer)
	}
	*/

	for _ , peer := range serv.config.PeerLst {
		//check if the new peer is in peer list
		if serv.IsExist(peer , false) {
			//fmt.Println("the Peer had existed ! peer = ",peer)
			continue
		}

		var msg = message {
			Src:      serv.addr,
			Dst:      peer,
			MsgType:  request,
		}

		req , err := json.Marshal(msg)
		if err != nil {
			return err
		}

		//fmt.Println("NetServer::ConnectSeeds want to connect peer = ",peer)
		go serv.Connect(peer , req , false)  //todo connect remote seed peer , if it's successful , add it into remote_list
	}

	return nil
}

//to connect specified peer
func (serv *NetServer) ConnectTo (conn net.Conn , msg []byte , isExist bool) error {
	fmt.Println("p2pServer::ConnectTo()")
	if conn == nil {
		return errors.New("*ERROR* Invalid parameter !!!")
	}

	_  , err := conn.Write(msg)
	if err != nil {
		fmt.Println("*ERROR* Failed to send data to the remote server addr !!! err: ",err)
		return err
	}

	return nil
}

//to connect to certain peer proactively
func (serv *NetServer) Connect(addr string , msg []byte , isExist bool) error {
	addr_port := addr+":"+fmt.Sprint(serv.port)
	//fmt.Println("p2pServer::Connect() addr_port = ",addr_port)

	conn , err := net.Dial("tcp", addr_port)
	if err != nil {
		//fmt.Println("*ERROR* Failed to create a connection for remote server !!! err: ",err)
		return err
	}

	len , err := conn.Write(msg)
	if err != nil {
		fmt.Println("*ERROR* Failed to send data to the remote server addr !!! err: ",err)
		return err
	}

	if len < 0 {
		fmt.Println("*ERROR* Failed to send data to the remote server addr !!! err: ",err)
		return err
	}

	fmt.Println("p2pServer::Connect() proactively finish to connect remote peer addr_port = ",addr_port," , len = ",len)

	return nil
}


//to connect certain peer with udp
func (serv *NetServer) ConnectUDP(addr string , msg []byte , isExist bool) error {
	fmt.Println("p2pServer::ConnectSeed() addr = ",addr)

	addr_port := addr+":"+fmt.Sprint(serv.port)
	remoteAddr, err := net.ResolveUDPAddr("udp4", addr_port)
	if err != nil {
		return errors.New("*ERROR* Failed to create a remote server addr !!!")
	}

	/*
	//test connection with remote peer
	var msg = message {
		src:      serv.addr,
		dst:      addr,
		msg_type: request,
	}

	req , err := json.Marshal(msg)
	if err != nil {
		return err
	}
	*/

	_ , err = serv.socket.WriteToUDP(msg , remoteAddr)
	if err != nil { //todo check len
		fmt.Println("*ERROR* Failed to send Test message to remote peer !!! ",err)
		return errors.New("*ERROR* Failed to send Test message to remote peer !!!")
	}

	/*
	//package remote peer info as "peer" struct and add it into peer list
	peer := NewPeer(addr)
	peer_identify := Hash(addr_port)
	serv.peerMap[uint64(peer_identify)] = peer
	*/

	return nil
}

func (serv *NetServer) IsExist(addr string , isExist bool) bool {
	//fmt.Println("NetServer::IsExist")
	for _ , peer := range serv.peerMap {
		if res := strings.Compare(peer.peerAddr , addr); res == 0 {
			//fmt.Println("NettrueServer::IsExist find")
			return true
		}
	}

	return false
}

func (serv *NetServer) WatchStatus() {
	fmt.Println("NetServer::WatchStatus")
	//for {
	//fmt.Println("<-------------------------------------------------------------->")
		for key, peer := range serv.peerMap {
			fmt.Println("<------------ NetServer::WatchStatus() key = ", key, " , peer = ", peer)
		}
	//}
}


func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}