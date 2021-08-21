package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strconv"

	"github.com/numary/numscript-ls/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type Server struct {
	reader *textproto.Reader
	files  map[lsp.DocumentURI]string
}

func NewServer() Server {
	return Server{
		reader: textproto.NewReader(bufio.NewReader(os.Stdin)),
		files:  make(map[lsp.DocumentURI]string),
	}
}

func Debug(format string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(format, args...))
}

func (s *Server) ReadRequest() jsonrpc2.Request {
	mime_header, err := s.reader.ReadMIMEHeader()
	if err != nil {
		os.Stderr.WriteString("ERROR:\n")
		os.Stderr.WriteString(err.Error())
	}

	len, err := strconv.ParseInt(mime_header["Content-Length"][0], 10, 0)
	if err != nil {
		os.Stderr.WriteString("ERROR:\n")
		os.Stderr.WriteString(err.Error())
	}

	buf := make([]byte, len)
	_, err = io.ReadFull(s.reader.R, buf)
	if err != nil {
		os.Stderr.WriteString("ERROR:\n")
		os.Stderr.WriteString(err.Error())
	}

	os.Stderr.WriteString("REQUEST: " + string(buf) + "\n")

	var req jsonrpc2.Request
	req.UnmarshalJSON(buf)
	return req
}

func (s *Server) SendResponse(res jsonrpc2.Response) {
	res_s, err := res.MarshalJSON()
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("unable to marshal JSON response: %v", err))
	}
	os.Stderr.WriteString("RESPONSE: " + string(res_s) + "\n")
	fmt.Printf("Content-Length: %v\r\n\r\n%v", len(res_s), string(res_s))
}

func (s *Server) Run() {
	for {
		req := s.ReadRequest()
		if handler, ok := handlers[req.Method]; ok {
			res, err := json.Marshal(handler(s, req.Params))
			if err != nil {
				panic(err)
			}
			if res != nil {
				res_raw := json.RawMessage(res)
				rpc_res := jsonrpc2.Response{
					ID:     req.ID,
					Result: &res_raw,
					Error:  nil,
					Meta:   nil,
				}
				s.SendResponse(rpc_res)
			}
		} else {
			os.Stderr.WriteString("unsupported method: " + req.Method + "\n")
		}
	}
}

func main() {
	Debug("Starting Numscript Language Server...\n")

	server := NewServer()

	server.Run()
}
