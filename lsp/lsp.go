package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"
	rpc2 "go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type Lsp struct {
	RootPath string
	Parser   *Parser
	RootConn rpc2.Conn
	Name     string
}

func DefaultLsp() *Lsp {
	return &Lsp{
		RootPath: "",
		Parser:   NewParser(),
		Name:     "lsp-template",
	}
}

func (lsp *Lsp) WalkFromRoot() {
	// exclude_dirs := []string{".git", "build", "vendor", "contrib"}
	filepath.WalkDir(
		lsp.RootPath,
		func(path string, d os.DirEntry, err error) error {
			return nil
		},
	)
}

func isPointInRange(needle sitter.Point, start_position sitter.Point, end_position sitter.Point) bool {
	return needle.Row >= start_position.Row &&
		needle.Row <= end_position.Row &&
		needle.Column >= start_position.Column &&
		needle.Column <= end_position.Column
}

func (lsp *Lsp) LspHandler(ctx context.Context, reply rpc2.Replier, req rpc2.Request) error {
	switch req.Method() {
	case protocol.MethodInitialize:
		params := req.Params()
		var replyParams protocol.InitializeParams
		err := json.Unmarshal(params, &replyParams)

		if err != nil {
			lsp.Log("cant unmarshal params", protocol.MessageTypeError)
			return reply(ctx, fmt.Errorf("cant unmarshal params"), nil)
		}
		path := replyParams.WorkspaceFolders[0].Name
		if path != "" {
			lsp.RootPath = path
		} else {
			ctx.Done()
			return reply(ctx, fmt.Errorf("no root path"), nil)
		}

		go func() {
			lsp.WalkFromRoot()
		}()

		return reply(
			ctx,
			protocol.InitializeResult{
				Capabilities: protocol.ServerCapabilities{
					TextDocumentSync:                 nil,
					CompletionProvider:               &protocol.CompletionOptions{},
					HoverProvider:                    true,
					SignatureHelpProvider:            &protocol.SignatureHelpOptions{},
					DeclarationProvider:              nil,
					DefinitionProvider:               nil,
					TypeDefinitionProvider:           nil,
					ImplementationProvider:           nil,
					ReferencesProvider:               nil,
					DocumentHighlightProvider:        nil,
					DocumentSymbolProvider:           nil,
					CodeActionProvider:               nil,
					CodeLensProvider:                 &protocol.CodeLensOptions{},
					DocumentLinkProvider:             &protocol.DocumentLinkOptions{},
					ColorProvider:                    nil,
					WorkspaceSymbolProvider:          nil,
					DocumentFormattingProvider:       nil,
					DocumentRangeFormattingProvider:  nil,
					DocumentOnTypeFormattingProvider: &protocol.DocumentOnTypeFormattingOptions{},
					RenameProvider:                   nil,
					FoldingRangeProvider:             nil,
					SelectionRangeProvider:           nil,
					ExecuteCommandProvider:           &protocol.ExecuteCommandOptions{},
					CallHierarchyProvider:            nil,
					LinkedEditingRangeProvider:       nil,
					SemanticTokensProvider:           nil,
					Workspace:                        &protocol.ServerCapabilitiesWorkspace{},
					MonikerProvider:                  nil,
					Experimental:                     nil,
				},
				ServerInfo: &protocol.ServerInfo{
					Name:    lsp.Name,
					Version: "0.0.0",
				},
			},
			nil,
		)

	case protocol.MethodTextDocumentHover:
		return reply(ctx, protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "hello from lsp-template",
			},
			Range: nil,
		}, nil)

	case protocol.MethodShutdown:
		// without this pylsp-test throws an error, but it's useless, i think
		return reply(ctx, fmt.Errorf("ShutDown"), nil)
	}
	// always return something otherwise other lsps responses can break
	// err shows up in the client as a popup/somewhere else in the UI in neovim
	// in the statusline
	// result is the result of the request
	return reply(ctx, fmt.Errorf("method not found: %q", req.Method()), nil)
}

func (lsp *Lsp) Log(message string, messageType protocol.MessageType) {
	// will send a message to the client, that will show up in :LspLog
	lsp.RootConn.Notify(context.Background(), protocol.MethodWindowLogMessage, protocol.LogMessageParams{
		Message: fmt.Sprintf("%s: %s", lsp.Name, message),
		Type:    messageType,
	})
}

func (lsp *Lsp) SendDiagnostic(path string, diagnostics *[]protocol.Diagnostic) {
	lsp.RootConn.Notify(context.Background(),
		protocol.MethodTextDocumentPublishDiagnostics,
		protocol.PublishDiagnosticsParams{
			URI:         uri.URI("file://" + path),
			Version:     0,
			Diagnostics: *diagnostics,
		},
	)
}

type rwc struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (rwc *rwc) Read(b []byte) (int, error)  { return rwc.r.Read(b) }
func (rwc *rwc) Write(b []byte) (int, error) { return rwc.w.Write(b) }
func (rwc *rwc) Close() error {
	rwc.r.Close()
	return rwc.w.Close()
}

func (lsp *Lsp) Init() {
	lsp = DefaultLsp()

	bufStream := rpc2.NewStream(&rwc{os.Stdin, os.Stdout})
	lsp.RootConn = rpc2.NewConn(bufStream)

	ctx := context.Background()
	lsp.RootConn.Go(ctx, lsp.LspHandler)
	<-lsp.RootConn.Done()
}
