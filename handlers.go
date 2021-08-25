package main

import (
	"encoding/json"
	"io/ioutil"

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

// func get_token_mod_field(name string) uint32 {
// 	for i, n := range SUPPORTED_TOKEN_MODS {
// 		if name == n {
// 			return 1 << uint32(i)
// 		}
// 	}
// 	return 0
// }

func compile(s *Server, uri lsp.DocumentURI, source string) {
	artifacts := compiler.CompileFull(source)
	errors := artifacts.Errors
	s.files[uri] = Document{
		content: source,
		tokens:  artifacts.Tokens,
		errors:  errors,
	}
	diagnostics := make([]lsp.Diagnostic, len(errors))
	for i, e := range errors {
		diagnostics[i] = lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      uint32(e.Startl - 1),
					Character: uint32(e.Startc),
				},
				End: lsp.Position{
					Line:      uint32(e.Endl - 1),
					Character: uint32(e.Endc),
				},
			},
			Severity: lsp.SeverityError,
			Message:  e.Msg,
		}
	}
	s.SendNotification("textDocument/publishDiagnostics", lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
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
				},
				SemanticTokensProvider: &lsp.SemanticTokensOptions{
					WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{},
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
				CompletionProvider: &lsp.CompletionOptions{
					TriggerCharacters:       []string{"s"},
					AllCommitCharacters:     []string{},
					ResolveProvider:         false,
					WorkDoneProgressOptions: lsp.WorkDoneProgressOptions{},
				},
			},
		}
	},

	"shutdown": func(s *Server, pr *json.RawMessage) interface{} {
		return map[string]struct{}{}
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
			compile(s, uri, string(text))
		}
		return nil
	},

	"textDocument/didChange": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.DidChangeTextDocumentParams
		json.Unmarshal([]byte(*pr), &p)
		uri := p.TextDocument.URI
		text := p.ContentChanges[len(p.ContentChanges)-1].Text
		compile(s, uri, string(text))
		return nil
	},

	"textDocument/semanticTokens/full": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.SemanticTokensParams
		json.Unmarshal([]byte(*pr), &p)
		uri := p.TextDocument.URI
		tokens := s.files[uri].tokens

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

			var token_type uint32
			switch token.GetTokenType() {
			case
				parser.NumScriptLexerTY_ACCOUNT,
				parser.NumScriptLexerTY_ASSET,
				parser.NumScriptLexerTY_NUMBER,
				parser.NumScriptLexerTY_MONETARY,
				parser.NumScriptLexerTY_PORTION:
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

	"textDocument/completion": func(s *Server, pr *json.RawMessage) interface{} {
		var p lsp.CompletionParams
		json.Unmarshal([]byte(*pr), &p)
		// uri := p.TextDocument.URI
		// range := p.TextDocu
		// pos := p.TextDocumentPositionParams.Position

		return lsp.CompletionList{
			IsIncomplete: false,
			Items: []lsp.CompletionItem{
				{
					Label:         "send!",
					Kind:          lsp.SnippetCompletion,
					Tags:          []lsp.CompletionItemTag{},
					Detail:        "auto-fill send",
					Documentation: "Send monetary value from a source to a destination.",
					InsertText: `send ${1:[CURRENCY 0]} (
	source = $2
	destination = $3
)`,
					InsertTextFormat:    lsp.SnippetTextFormat,
					AdditionalTextEdits: []lsp.TextEdit{},
					CommitCharacters:    []string{},
					Command:             &lsp.Command{},
					Data:                nil,
				},
				{
					Label:         "vars!",
					Kind:          lsp.SnippetCompletion,
					Tags:          []lsp.CompletionItemTag{},
					Detail:        "auto-fill vars",
					Documentation: "Declare variables of the script.",
					InsertText: `vars {
	$1 \$$2
}`,
					InsertTextFormat:    lsp.SnippetTextFormat,
					AdditionalTextEdits: []lsp.TextEdit{},
					CommitCharacters:    []string{},
					Command:             &lsp.Command{},
					Data:                nil,
				},
			},
		}
	},
}

var notification_handlers = map[string]func(*Server, *json.RawMessage){
	"initialized": func(s *Server, pr *json.RawMessage) {},
}
