package util

import (
	"github.com/lmittmann/tint"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigureSlog so that it easy to locate the source file & line as the Goland IDE picks up the relative file path.
// TODO: Remove that when you're done
func ConfigureSlog(writeTo io.Writer) {
	wd, err := os.Getwd()
	var tintHandler slog.Handler
	if err != nil {
		slog.Error("Unable to find working dir, falling back to default slog Config")
		tintHandler = tint.NewHandler(writeTo, &tint.Options{AddSource: true})
	} else {
		unixPath := filepath.ToSlash(wd)
		tintHandler = tint.NewHandler(writeTo, &tint.Options{
			AddSource: true,
			ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
				if attr.Key == slog.SourceKey {
					source, ok := attr.Value.Any().(*slog.Source)
					relativePath := "." + strings.TrimPrefix(source.File, unixPath)
					var sb strings.Builder
					sb.WriteString(relativePath)
					sb.WriteString(":")
					sb.WriteString(strconv.Itoa(source.Line))
					if !ok {
						panic("Unable to assert type on source attr while configuring tint handler")
					}
					return slog.Attr{
						Key:   attr.Key,
						Value: slog.StringValue(sb.String()),
					}
				}
				return attr
			},
		})
	}
	slog.SetDefault(slog.New(tintHandler))
}
