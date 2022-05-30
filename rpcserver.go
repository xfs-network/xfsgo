// Copyright 2018 The xfsgo Authors
// This file is part of the xfsgo library.
//
// The xfsgo library is free software: you can redistribute it and/or modify
// it under the terms of the MIT Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The xfsgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// MIT Lesser General Public License for more details.
//
// You should have received a copy of the MIT Lesser General Public License
// along with the xfsgo library. If not, see <https://mit-license.org/>.

package xfsgo

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go/token"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"strings"
	"xfsgo/log"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	jsonrpcVersion = "2.0"
)

type RPCConn interface {
	SendMessage(uuid.UUID, interface{}) error
	SetCloseHandler(h func(code int, text string) error)
}
type rpcConn struct {
	conn    *websocket.Conn
	request *RPCMessageRequest
}

func (c *rpcConn) SendMessage(id uuid.UUID, data interface{}) error {
	msg := &RPCBroadcastMsg{
		Jsonrpc:      jsonrpcVersion,
		Id:           c.request.Id,
		Result:       data,
		Subscription: id.String(),
	}
	return sendWSRPCResponse(c.conn, msg)
}

func (c *rpcConn) SetCloseHandler(h func(code int, text string) error) {
	if h != nil {
		c.conn.SetCloseHandler(h)
	}
}

type method struct {
	method    reflect.Method
	argc      uint
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name         string
	rcvr         reflect.Value
	typ          reflect.Type
	methods      map[string]*method
	isSubscriber bool
}

type RPCConfig struct {
	ListenAddr string
	Logger     log.Logger
}

// RPCServer is an RPC server.
type RPCServer struct {
	logger     log.Logger
	config     *RPCConfig
	ginEngine  *gin.Engine
	upgrader   websocket.Upgrader
	serviceMap map[string]*service
}

func ginlogger(log log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(c.Errors) > 0 {
			log.Errorln(c.Errors.ByType(gin.ErrorTypePrivate).String())
		}
	}
}

func ginCors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func NewRPCServer(config *RPCConfig) *RPCServer {
	server := &RPCServer{
		logger:     config.Logger,
		config:     config,
		serviceMap: make(map[string]*service),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	if server.logger == nil {
		server.logger = log.DefaultLogger()
	}
	gin.DefaultWriter = server.logger.Writer()
	gin.SetMode("release")
	server.ginEngine = gin.New()
	server.ginEngine.Use(ginlogger(server.logger))
	server.ginEngine.Use(gin.Recovery())
	server.ginEngine.Use(ginCors())
	return server
}

// RegisterName creates a service for the given receiver type under the given name and added it to the
// service collection this server provides to clients.
func (server *RPCServer) RegisterName(name string, rcvr interface{}) error {
	return server.register(rcvr, name, true, false)
}
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

var typeOfRPCConn = reflect.TypeOf((*RPCConn)(nil)).Elem()

func suitableMethods(objType reflect.Type, isSubscriber bool) map[string]*method {
	methods := make(map[string]*method)
	for i := 0; i < objType.NumMethod(); i++ {
		methodN := objType.Method(i)
		mtype := methodN.Type
		mname := methodN.Name
		if !methodN.IsExported() {
			continue
		}
		if mtype.NumIn() != 4 && isSubscriber {
			continue
		} else if !isSubscriber && mtype.NumIn() != 3 {
			continue
		}
		var argType reflect.Type
		var replyType reflect.Type
		if isSubscriber {
			if connType := mtype.In(1); connType != typeOfRPCConn {
				continue
			}
			argType = mtype.In(2)
			replyType = mtype.In(3)
		} else {
			argType = mtype.In(1)
			replyType = mtype.In(2)
		}
		if !isExportedOrBuiltinType(argType) {
			continue
		}
		if replyType.Kind() != reflect.Ptr {
			continue
		}
		if !isExportedOrBuiltinType(replyType) {
			continue
		}
		if mtype.NumOut() != 1 {
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfError {
			continue
		}
		methods[mname] = &method{
			method:    methodN,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}
	return methods

}
func (server *RPCServer) register(rcvr interface{}, name string, useName bool, isSubscriber bool) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := name
	if !useName {
		sname = reflect.Indirect(s.rcvr).Type().Name()
	}
	if sname == "" {
		return fmt.Errorf("rpc register: no service name for type %s", s.typ.String())
	}
	if !useName && !token.IsExported(sname) {
		return fmt.Errorf("rpc register: type %s is not exported", sname)
	}
	s.name = sname
	s.isSubscriber = isSubscriber
	s.methods = suitableMethods(s.typ, isSubscriber)
	server.serviceMap[s.name] = s
	return nil
}
func (server *RPCServer) RegisterSubscribe(name string, obj interface{}) error {
	return server.register(obj, name, true, true)

}

func isWebsocketRequest(c *gin.Context) bool {
	connection := c.GetHeader("Connection")
	upgrade := c.GetHeader("Upgrade")
	return connection == "Upgrade" && upgrade == "websocket"
}
func (server *RPCServer) handleWebsocket(c *gin.Context) error {
	server.upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	conn, err := server.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Warnf("upgrad err: %s", err)
		return err
	}
	server.readLoop(conn)
	return nil
}

type RPCMessageRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RPCMessageRespSuccess struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

type RPCMessageError struct {
	Jsonrpc string              `json:"jsonrpc"`
	Id      interface{}         `json:"id"`
	Error   *RPCMessageErrorObj `json:"error"`
}

type RPCBroadcastMsg struct {
	Jsonrpc      string      `json:"jsonrpc"`
	Id           interface{} `json:"id"`
	Result       interface{} `json:"result"`
	Subscription string      `json:"subscription"`
}

type RPCMessageErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func sendRPCResponse(writer io.WriteCloser, resp interface{}) error {
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = writer.Write(raw)
	if err != nil {
		return err
	}
	return writer.Close()
}

func sendWSRPCResponse(conn *websocket.Conn, resp interface{}) error {
	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	return sendRPCResponse(w, resp)
}

func sendWSRPCError(conn *websocket.Conn, id interface{}, err error) error {
	errobj := packErrorMessage(id, err)
	return sendWSRPCResponse(conn, errobj)
}
func (server *RPCServer) readLoop(conn *websocket.Conn) {
	defer func() {
		if conn == nil {
			return
		}
		if err := conn.Close(); err != nil {
			logrus.Warnf("websocket close err: %s", err)
		}
	}()
	conn.SetPingHandler(nil)
	conn.SetPongHandler(nil)
	conn.SetCloseHandler(nil)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var request *RPCMessageRequest
		if err = json.Unmarshal(data, &request); err == nil {
			var replay interface{}
			if err = server.gotRPCRequestReply(request, &replay, &rpcConn{
				request: request,
				conn:    conn,
			}); err != nil {
				_ = sendWSRPCError(conn, request.Id, err)
				continue
			}
			data := &RPCMessageRespSuccess{
				Jsonrpc: jsonrpcVersion,
				Id:      request.Id,
				Result:  replay,
			}
			_ = sendWSRPCResponse(conn, data)
			continue
		}
		_ = sendWSRPCError(conn, nil, parseError)
	}
}
func sendHTTPRPCResponse(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}

func packErrorMessage(id interface{}, err error) *RPCMessageError {
	var errorCode int
	var errorMessage string
	if err == nil {
		errorCode = internalError.Code
		errorMessage = internalError.Message
	} else if rpcerr, ok := err.(*rpcError); ok {
		errorCode = rpcerr.Code
		errorMessage = rpcerr.Message
	} else {
		errorCode = internalError.Code
		errorMessage = err.Error()
	}
	msg := &RPCMessageError{
		Jsonrpc: jsonrpcVersion,
		Id:      id,
		Error:   &RPCMessageErrorObj{Code: errorCode, Message: errorMessage},
	}
	return msg
}
func sendHTTPRPCError(c *gin.Context, code int, id interface{}, err error) {
	msg := packErrorMessage(id, err)
	sendHTTPRPCResponse(c, code, msg)
}

//Start starts rpc server.
func (server *RPCServer) Start() error {
	server.ginEngine.Any("/", func(c *gin.Context) {
		//handle websocket request
		defer c.Abort()
		if isWebsocketRequest(c) {
			if err := server.handleWebsocket(c); err != nil {
				server.logger.Warnf("ws connect err")
			}
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		if "POST" != c.Request.Method {
			sendHTTPRPCError(c, 400, nil, invalidRequestError)
			return
		}
		contentType := c.ContentType()
		if contentType != "application/json" {
			sendHTTPRPCError(c, 400, nil, invalidRequestError)
			return
		}
		if nil == c.Request.Body {
			sendHTTPRPCError(c, 400, nil, invalidRequestError)
			return
		}
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			sendHTTPRPCError(c, 400, nil, parseError)
			return
		}
		var request *RPCMessageRequest
		if err = json.Unmarshal(body, &request); err == nil {
			var reply interface{}
			if err = server.gotRPCRequestReply(request, &reply, nil); err != nil {
				sendHTTPRPCError(c, 200, request.Id, err)
				return
			}
			data := &RPCMessageRespSuccess{
				Jsonrpc: jsonrpcVersion,
				Id:      request.Id,
				Result:  reply,
			}
			sendHTTPRPCResponse(c, 200, data)
			return
		}
        var batchs []*RPCMessageRequest
        if err = json.Unmarshal(body, &batchs); err == nil {
            var resps []interface{}
            resps = make([]interface{}, len(batchs))
            for i := 0; i < len(batchs); i++ {
                requestSigle := batchs[i]
                var reply interface{}
                var resp interface{}
                if err = server.gotRPCRequestReply(requestSigle, &reply, nil); err != nil {
                    resp = packErrorMessage(requestSigle.Id, err)
                }else {
                    resp = &RPCMessageRespSuccess{
                        Jsonrpc: jsonrpcVersion,
                        Id:      request.Id,
                        Result:  reply,
                    }
                }
                resps[i] = resp;
            }
            sendHTTPRPCResponse(c, 200, resps)
            return
        }
		sendHTTPRPCError(c, 400, nil, parseError)
	})

	ln, err := net.Listen("tcp", server.config.ListenAddr)
	if err != nil {
		return err
	}
	server.logger.Infof("RPC Service listen on: %s", ln.Addr())
	return server.ginEngine.RunListener(ln)
}
func (server *RPCServer) readRequest(request *RPCMessageRequest) (
	s *service, m *method, argv reflect.Value, replyv reflect.Value, err error) {
	if request.Method == "" {
		err = methodNotFoundError
		return
	}
	dot := strings.LastIndex(request.Method, ".")
	if dot < 0 {
		err = methodNotFoundError
		return
	}
	serviceName := request.Method[:dot]
	methodName := request.Method[dot+1:]
	s, existsService := server.serviceMap[serviceName]
	if !existsService {
		err = methodNotFoundError
		return
	}
	m, existsMethod := s.methods[methodName]
	if !existsMethod {
		err = methodNotFoundError
		return
	}
	argTypeKind := m.ArgType.Kind()
	argIsValue := false
	if argTypeKind == reflect.Ptr {
		argv = reflect.New(m.ArgType.Elem())
	} else {
		argv = reflect.New(m.ArgType)
		argIsValue = true
	}
	err = json.Unmarshal(request.Params, argv.Interface())
	if err != nil {
		var params []interface{}
		err = json.Unmarshal(request.Params, &params)
		n := m.ArgType.NumField()
		if len(params) != n {
			return
		}
		for i := 0; i < n; i++ {
			var field reflect.Value
			if argIsValue {
				field = argv.Elem().Field(i)
			} else {
				field = argv.Field(i)
			}
			field.Set(reflect.ValueOf(params[i]))
		}
		err = nil
	}
	if argIsValue {
		argv = argv.Elem()
	}
	replyv = reflect.New(m.ReplyType.Elem())

	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return
}

func (s *service) call(
	methodType *method,
	argv reflect.Value,
	replyv reflect.Value,
	reply *interface{},
	conn *rpcConn) error {
	function := methodType.method.Func

	var args []reflect.Value
	if s.isSubscriber && conn != nil {
		connval := reflect.ValueOf(conn)
		args = []reflect.Value{s.rcvr, connval, argv, replyv}
	} else if s.isSubscriber && conn == nil {
		return internalError
	} else {
		args = []reflect.Value{s.rcvr, argv, replyv}
	}
	returnValue := function.Call(args)
	errInter := returnValue[0].Interface()
	replyInter := replyv.Interface()
	*reply = replyInter
	if returnValue[0].IsNil() {
		return nil
	}
	return errInter.(error)
}
func (server *RPCServer) gotRPCRequestReply(request *RPCMessageRequest, reply *interface{}, conn *rpcConn) error {
	s, m, argv, replyv, err := server.readRequest(request)
	if err != nil {
		return err
	}
	return s.call(m, argv, replyv, reply, conn)
}
