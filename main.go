package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strconv"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/numary/machine/script/compiler"
	"github.com/numary/numscript-ls/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	reader *textproto.Reader
	out    chan string
	files  map[lsp.DocumentURI]Document
}

type Document struct {
	content string
	tokens  []antlr.Token
	errors  []compiler.CompileError
}

func NewServer() Server {
	out := make(chan string)

	go func() {
		for res := range out {
			Debug("RESPONSE: " + res + "\n")
			fmt.Printf("Content-Length: %v\r\n\r\n%v", len(res), res)
		}
	}()

	return Server{
		reader: textproto.NewReader(bufio.NewReader(os.Stdin)),
		out:    out,
		files:  make(map[lsp.DocumentURI]Document),
	}
}

func Debug(format string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(format, args...))
}

func (s *Server) ReadRequest() jsonrpc2.Request {
	mime_header, err := s.reader.ReadMIMEHeader()
	if err != nil {
		if err.Error() == "EOF" {
			os.Exit(0)
		}
		panic(err)
	}

	len, err := strconv.ParseInt(mime_header["Content-Length"][0], 10, 0)
	if err != nil {
		panic(err)
	}

	buf := make([]byte, len)
	_, err = io.ReadFull(s.reader.R, buf)
	if err != nil {
		panic(err)
	}

	Debug("REQUEST: " + string(buf) + "\n\n")

	var req jsonrpc2.Request
	req.UnmarshalJSON(buf)
	return req
}

func (s *Server) SendResponse(msg interface{}, id jsonrpc2.ID) {
	res, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	res_raw := json.RawMessage(res)
	rpc_res := jsonrpc2.Response{
		ID:     id,
		Result: &res_raw,
		Error:  nil,
		Meta:   nil,
	}

	res_s, err := rpc_res.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("unable to marshal JSON response: %v", err))
	}
	Debug("RESPONSE: " + string(res_s) + "\n")
	fmt.Printf("Content-Length: %v\r\n\r\n%v", len(res_s), string(res_s))
}

func (s *Server) SendNotification(method string, params interface{}) {
	params_json, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	params_raw := json.RawMessage(params_json)
	rpc_notif := jsonrpc2.Request{
		Notif:  true,
		Method: method,
		Params: &params_raw,
	}

	res_s, err := rpc_notif.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("unable to marshal JSON response: %v", err))
	}
	Debug("RESPONSE: " + string(res_s) + "\n")
	fmt.Printf("Content-Length: %v\r\n\r\n%v", len(res_s), string(res_s))
}

func (s *Server) Run() {
	for {
		req := s.ReadRequest()
		if req.Method == "exit" {
			break
		}
		if handler, ok := handlers[req.Method]; ok {
			s.SendResponse(handler(s, req.Params), req.ID)
		} else if handler, ok := notification_handlers[req.Method]; ok {
			handler(s, req.Params)
		} else {
			Debug("unsupported method: " + req.Method + "\n")
		}
	}
}

func main() {
	Debug("Starting Numscript Language Server...\n")

	server := NewServer()

	server.Run()
}
