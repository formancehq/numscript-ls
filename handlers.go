package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/antlr/antlr4/runtime/Go/antlr"
	"github.com/numary/machine/script/compiler"
	"github.com/numary/machine/script/parser"
	"github.com/numary/numscript-ls/lsp"
)

var SUPPORTED_TOKEN_TYPES []string = []string{
	"comment",
	"keyword",
	"string",
	"number",
	"function",
	"type",
	"variable",
	"parameter",
}

func get_token_idx(name string) uint32 {
	for i, n := range SUPPORTED_TOKEN_TYPES {
		if name == n {
			return uint32(i)
		}
	}
	return 0
}

var SUPPORTED_TOKEN_MODS []string = []string{
	"deprecated",
}

func get_token_mod_field(name string) uint32 {
	for i, n := range SUPPORTED_TOKEN_MODS {
		if name == n {
			return 1 << uint32(i)
		}
	}
	return 0
}

var handlers = map[string]func(*Server, *json.RawMessage) interface{}{
	"initialize": func(s *Server, p *json.RawMessage) interface{} {
		return lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: &lsp.TextDocumentSyncOptions{
					OpenClose:         true,
					Change:            lsp.Full,
					WillSave:          false,
					WillSaveWaitUntil: false,
					Save: lsp.SaveOptions{
						IncludeText: false,
					},
				},
				SemanticTokensProvider: &lsp.SemanticTokensOptions{
					WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{
						WorkDoneProgress: true,
					},
					Legend: lsp.SemanticTokensLegend{
						TokenTypes:     SUPPORTED_TOKEN_TYPES,
						TokenModifiers: SUPPORTED_TOKEN_MODS,
					},
					Range: false,
					Full: &lsp.SemanticTokensOptions{
						Legend:                  lsp.SemanticTokensLegend{},
						Range:                   false,
						Full:                    nil,
						WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{},
					},
				},
			},
		}
	},

	"textDocument/didOpen": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.DidOpenTextDocumentParams
		json.Unmarshal([]byte(*pr), &p)
		uri := p.TextDocument.URI
		if uri[0:7] == "file://" {
			path := string(uri[7:])
			text, err := ioutil.ReadFile(path)
			if err != nil {
				panic("could not open file: " + err.Error())
			}
			s.files[uri] = string(text)
		}
		return nil
	},

	"textDocument/didChange": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.DidChangeTextDocumentParams
		json.Unmarshal([]byte(*pr), &p)
		uri := p.TextDocument.URI
		s.files[uri] = p.ContentChanges[len(p.ContentChanges)-1].Text
		os.Stderr.WriteString(s.files[uri])
		return nil
	},

	"textDocument/semanticTokens/full": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.SemanticTokensParams
		json.Unmarshal([]byte(*pr), &p)
		uri := p.TextDocument.URI
		text, ok := s.files[uri]
		if !ok {
			return nil
		}

		elistener := &compiler.ErrorListener{}
		is := antlr.NewInputStream(text)
		lexer := parser.NewNumScriptLexer(is)
		lexer.RemoveErrorListeners()
		lexer.AddErrorListener(elistener)
		stream := antlr.NewCommonTokenStream(lexer, antlr.LexerDefaultTokenChannel)
		pars := parser.NewNumScriptParser(stream)
		pars.RemoveErrorListeners()
		pars.AddErrorListener(elistener)
		pars.Script()
		tokens := stream.GetAllTokens()

		out := []uint32{}
		line := 1
		column := 0
		for i := 0; i < len(tokens); i++ {
			token := tokens[i]
			l := token.GetLine()
			c := token.GetColumn()
			length := uint32(len(token.GetText()))
			delta_line := uint32(l - line)
			var delta_start uint32
			if l == line {
				delta_start = uint32(c) - uint32(column)
			} else {
				delta_start = uint32(c)
			}

			Debug("Token: %-6s, DL: %v, DC: %v\n", token.GetText(), delta_line, delta_start)

			var token_type uint32
			switch token.GetTokenType() {
			case
				parser.NumScriptLexerTY_ACCOUNT,
				parser.NumScriptLexerTY_ASSET,
				parser.NumScriptLexerTY_NUMBER,
				parser.NumScriptLexerTY_MONETARY:
				token_type = get_token_idx("type")
			case
				parser.NumScriptLexerVARS,
				parser.NumScriptLexerSEND,
				parser.NumScriptLexerSOURCE,
				parser.NumScriptLexerFROM,
				parser.NumScriptLexerMAX,
				parser.NumScriptLexerDESTINATION,
				parser.NumScriptLexerTO:
				token_type = get_token_idx("keyword")
			case
				parser.NumScriptLexerVARIABLE_NAME:
				token_type = get_token_idx("variable")
			case
				parser.NumScriptLexerSTRING:
				token_type = get_token_idx("string")
			case
				parser.NumScriptLexerACCOUNT,
				parser.NumScriptLexerASSET,
				parser.NumScriptLexerNUMBER,
				parser.NumScriptLexerPORTION:
				token_type = get_token_idx("number")
			case parser.NumScriptLexerMETA:
				token_type = get_token_idx("function")
			default:
				continue
			}
			out = append(out, delta_line, delta_start, length, token_type, 0)
			line = l
			column = c
		}
		return lsp.SemanticTokens{
			Data: out,
		}
	},
}
