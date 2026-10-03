package commands

import (
	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/output"
)

// apiBody is a RequestOption convenience for JSON request bodies.
func apiBody(v any) api.RequestOption { return api.WithBody(v) }

func timeAgo(ts string) string { return output.TimeAgo(ts) }
