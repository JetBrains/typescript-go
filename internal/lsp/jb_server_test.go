package lsp

import (
	"context"
	"io"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/lsp/lsproto"
	"github.com/microsoft/typescript-go/internal/project"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
)

func TestGetProjectAndFileNameIgnoresInvalidProjectFileNameWhenDefaultProjectExists(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled files are not embedded")
	}

	ctx, cancel := context.WithCancel(context.Background())
	fs := bundled.WrapFS(vfstest.FromMap(map[string]string{
		"/test/tsconfig.json": "{}",
		"/test/index.ts":      "const x = 1;",
	}, false))

	server := NewServer(&ServerOptions{
		In:                 shutdownTestReader{},
		Out:                shutdownTestWriter{},
		Err:                io.Discard,
		Cwd:                "/test",
		FS:                 fs,
		DefaultLibraryPath: bundled.LibPath(),
	})
	server.backgroundCtx = ctx
	server.session = project.NewSession(&project.SessionInit{
		BackgroundCtx: ctx,
		Options: &project.SessionOptions{
			CurrentDirectory:   "/test",
			DefaultLibraryPath: bundled.LibPath(),
			PositionEncoding:   lsproto.PositionEncodingKindUTF8,
			WatchEnabled:       false,
			LoggingEnabled:     false,
		},
		FS:     fs,
		Logger: server.logger,
	})
	t.Cleanup(func() {
		cancel()
		server.session.Close()
	})

	projectFileName := lsproto.DocumentUri("file:///tsconfig.json")
	project, fileName, err := server.GetProjectAndFileName(&projectFileName, "file:///test/index.ts", ctx)

	assert.NilError(t, err)
	assert.Assert(t, project != nil)
	assert.Equal(t, project.Name(), "/test/tsconfig.json")
	assert.Equal(t, fileName, "/test/index.ts")
}
