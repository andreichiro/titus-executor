package docker

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	runtimeTypes "github.com/Netflix/titus-executor/executor/runtime/types"
	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/alessio/shellescape.v1"
)

const envFileTemplateStr = `# This file was autogenerated by the titus executor

{{ range $key, $val := .ContainerEnv -}}
export {{ $key }}={{ $val | escape_sq }}
{{ end -}}

{{ if (.ImageEnv | len) gt 0 }}
# These environment variables were in your docker configuration
{{ range $key, $val := .ImageEnv -}}
export {{ $key }}={{ escapeSQWithFallback $key $val }}
{{ end -}}
{{ end -}}
`

var (
	funcMap         = template.FuncMap{"escape_sq": shellescape.Quote, "escapeSQWithFallback": escapeSQWithFallback}
	envFileTemplate = template.Must(template.New("").Funcs(funcMap).Parse(envFileTemplateStr))
)

// See here: https://wiki.bash-hackers.org/syntax/pe#use_a_default_value
// This only works under bash AFAIK.
func escapeSQWithFallback(key, fallBackValue string) string {
	// Key does not need to be escaped, because it's always safe
	return fmt.Sprintf("${%s-%s}", key, shellescape.Quote(fallBackValue))
}

type envFileTemplateData struct {
	ContainerEnv map[string]string
	ImageEnv     map[string]string
}

func executeEnvFileTemplate(c *runtimeTypes.Container, imageInfo *types.ImageInspect, buf io.Writer) error {
	imageEnv := make(map[string]string, len(imageInfo.Config.Env))

	for _, environmentVariable := range imageInfo.Config.Env {
		splitEnvironmentVariable := strings.SplitN(environmentVariable, "=", 2)
		if len(splitEnvironmentVariable) != 2 {
			logrus.WithField("environmentVariable", environmentVariable).Warning("Cannot parse environment variable")
			continue
		}
		imageEnv[splitEnvironmentVariable[0]] = splitEnvironmentVariable[1]
	}

	templateData := envFileTemplateData{ContainerEnv: c.Env, ImageEnv: imageEnv}
	return envFileTemplate.Execute(buf, templateData)
}
