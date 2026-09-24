package lsp

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/lsp/uri"
	"github.com/open-policy-agent/regal/internal/ogre"
	"github.com/open-policy-agent/regal/pkg/config"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
	"github.com/open-policy-agent/regal/pkg/roast/transform"
)

var (
	emptyLocations any = []any{}

	testLocationsQuery = sync.OnceValue(func() *ogre.Query {
		query := ast.MustParseBody("result = data.regal.lsp.testlocations.result")

		pq, err := ogre.New(query).Prepare(context.Background())
		if err != nil {
			panic("failed to prepare test locations query: " + err.Error())
		}

		return pq
	})
)

func (l *LanguageServer) StartTestLocationsWorker(ctx context.Context) {
	l.workersWg.Go(func() {
		// Wait for initialization to complete before starting to check worker is needed
		<-l.initializationGate

		if !l.Workspace().Client().InitOptions.EnableServerTesting {
			l.log.Debug("Test locations worker exiting - client does not support opaTestProvider")

			return
		}

		l.log.Debug("Test locations worker starting")

		for {
			select {
			case <-ctx.Done():
				return
			case job := <-l.testLocationJobs:
				if err := l.processTestLocationsUpdate(ctx, job.URI); err != nil {
					l.log.Message("failed to process test locations: %s", err)
				}
			}
		}
	})
}

// processTestLocationsUpdate queries for test rule locations and sends them to the client.
// This is called after the parse has completed in the lintFileJobs worker to
// ensure we send the latest.
func (l *LanguageServer) processTestLocationsUpdate(ctx context.Context, fileURI string) error {
	if !l.Workspace().Client().InitOptions.EnableServerTesting {
		l.log.Message("processTestLocationsUpdate called but client does not support opaTestProvider")

		return nil
	}

	if l.ignoreURI(fileURI) {
		return nil
	}

	if ok := l.cache.HasFileContents(fileURI); !ok {
		return nil
	}

	contents, ok := l.cache.GetFileContents(fileURI)
	if !ok {
		return fmt.Errorf("test locations: failed to get file contents for uri %q", fileURI)
	}

	module, ok := l.cache.GetModule(fileURI)
	if !ok {
		// This shouldn't happen since parsing completed successfully before
		// calling
		return l.sendTestLocations(ctx, fileURI, emptyLocations)
	}

	// TODO: ideally this would not need fs access
	roots, err := config.GetPotentialRoots(uri.ToPath(fileURI))
	if err != nil || len(roots) == 0 {
		return fmt.Errorf("failed to get roots for file: %s", fileURI)
	}

	// the root is just returned verbatim, however this is used to group
	// packages in the UI, since the same package can appear in many roots.
	root := slices.MaxFunc(roots, func(a, b string) int {
		return len(a) - len(b)
	})

	relativePath := l.Workspace().RelativePath(fileURI)
	regalContext, _ := transform.RegalContext(relativePath, contents, module.RegoVersion().String()).Merge(
		ast.NewObject(rast.Item("file", ast.ObjectTerm(rast.Item("root", ast.StringTerm(root))))))

	astValue, err := transform.ToASTWithRegalContext(module, regalContext)
	if err != nil {
		l.log.Message("failed to prepare AST for test locations: %s", err)

		return l.sendTestLocations(ctx, fileURI, emptyLocations)
	}

	return testLocationsQuery().Evaluator().
		WithInput(astValue).
		WithResultHandler(ogre.ResultHandlerFunc(func(r ogre.Result) error {
			nativeResult, err := ast.JSON(r.Value)
			if err != nil {
				return l.sendTestLocations(ctx, fileURI, emptyLocations)
			}

			return l.sendTestLocations(ctx, fileURI, nativeResult)
		})).
		Eval(ctx)
}

func (l *LanguageServer) sendTestLocations(ctx context.Context, fileURI string, locations any) error {
	if l.conn == nil {
		l.log.Debug("sendTestLocations called with no connection: %s", fileURI)

		return nil
	}

	params := map[string]any{
		"uri":       fileURI,
		"locations": locations,
	}

	if err := l.conn.Notify(ctx, "regal/testLocations", params); err != nil {
		return fmt.Errorf("failed to send test locations notification: %w", err)
	}

	return nil
}
