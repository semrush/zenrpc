package main

import (
	"github.com/semrush/zenrpc/parser"
	"text/template"
)

var (
	serviceTemplate = template.Must(template.New("service").
		Funcs(template.FuncMap{"definitions": parser.Definitions}).
		Parse(`
{{define "smdType" -}}
	Type: smd.{{.Type}},
	{{- if eq .Type "Array" }}
		Items: map[string]string{
			{{- if eq .ItemsType "Object" }}
			"$ref": "#/definitions/{{.Ref}}",
			{{else}}
			"type": smd.{{.ItemsType}},
			{{end}}
		},
	{{- end}}
{{- end}}

{{define "properties" -}}
	Properties: map[string]smd.Property{
	{{range $i, $e := . -}}
		"{{.Name}}": {
			Description: ` + "`{{.Description}}`" + `,
			{{- if eq .SMDType.Type "Object" }}
				Ref: "#/definitions/{{.SMDType.Ref}}",
			{{- end}}			
			{{template "smdType" .SMDType}}
		},
	{{ end }}
	},
{{- end}}

{{define "definitions" -}}
	{{if .}}
	Definitions: map[string]smd.Definition{
		{{- range $name, $struct := .}}
			"{{ $name}}": {
				{{ template "properties" .Properties}}
			},
		{{ end }}
	},
	{{ end }}
{{- end}}


// Code generated by zenrpc; DO NOT EDIT.

package {{.PackageName}}

import (
	"encoding/json"
	"context"

	"github.com/semrush/zenrpc"
	"github.com/semrush/zenrpc/smd"

	{{ range .Imports}}
		{{if .Name}}{{.Name.Name}} {{end}}{{.Path.Value}}
	{{- end }}
)

var RPC = struct {
{{ range .Services}}
	{{.Name}} struct { {{range $i, $e := .Methods }}{{if $i}}, {{end}}{{.Name}}{{ end }} string } 
{{- end }}
}{	
	{{- range .Services}}
		{{.Name}}: struct { {{range $i, $e := .Methods }} {{if $i}}, {{end}}{{.Name}}{{ end }} string }{ 
			{{- range .Methods }}
				{{.Name}}:   "{{.LowerCaseName}}",
			{{- end }}
		}, 	
	{{- end }}
}

{{ range $s := .Services}}

	func ({{.Name}}) SMD() smd.ServiceInfo {
		return smd.ServiceInfo{
			Description: ` + "`{{.Description}}`" + `,
			Methods: map[string]smd.Service{ 
				{{- range .Methods }}
					"{{.Name}}": {
						Description: ` + "`{{.Description}}`" + `,
						Parameters: []smd.JSONSchema{ 
						{{- range .Args }}
							{
								Name: "{{.Name}}",
								Optional: {{.HasStar}},
								Description: ` + "`{{.Description}}`" + `,
								{{template "smdType" .SMDType}}
								{{- if and (eq .SMDType.Type "Object") (ne .SMDType.Ref "")}}
									{{ template "properties" (index $.Structs .SMDType.Ref).Properties}}
								{{- end}}
								{{- template "definitions" definitions .SMDType $.Structs }}
							},
						{{- end }}
						}, 
						{{- if .SMDReturn}}
							Returns: smd.JSONSchema{ 
								Description: ` + "`{{.SMDReturn.Description}}`" + `,
								Optional:    {{.SMDReturn.HasStar}},
								{{template "smdType" .SMDReturn.SMDType }}
								{{- if and (eq .SMDReturn.SMDType.Type "Object") (ne .SMDReturn.SMDType.Ref "")}}
									{{ template "properties" (index $.Structs .SMDReturn.SMDType.Ref).Properties}}
								{{- end}}
								{{- template "definitions" definitions .SMDReturn.SMDType $.Structs }}							
							}, 
						{{- end}}
						{{- if .Errors}}
							Errors: map[int]string{
								{{- range .Errors }}
									{{.Code}}: "{{.Description}}",
								{{- end }}
							},
						{{- end}}
					}, 
				{{- end }}
			},
		}
	}

	// Invoke is as generated code from zenrpc cmd
	func (s {{.Name}}) Invoke(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
		resp := zenrpc.Response{}
		var err error

		switch method { 
		{{- range .Methods }}
			case RPC.{{$s.Name}}.{{.Name}}: {{ if .Args }}
					var args = struct {
						{{ range .Args }}
							{{.CapitalName}} {{.Type}} ` + "`json:\"{{.JsonName}}\"`" + ` 
						{{- end }}
					}{}

					if zenrpc.IsArray(params) {
						if params, err = zenrpc.ConvertToObject([]string{ 
							{{- range .Args }}"{{.JsonName}}",{{ end -}} 
							}, params); err != nil {
							return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
						}
					}

					if len(params) > 0 {
						if err := json.Unmarshal(params, &args); err != nil {
							return zenrpc.NewResponseError(nil, zenrpc.InvalidParams, err.Error(), nil)
						}
					}

					{{ range .DefaultValues }}
						{{.Comment}}
						if args.{{.CapitalName}} == nil {
							var v {{.Type}} = {{.Value}}
							args.{{.CapitalName}} = &v
						}
					{{ end }}

				{{ end }} {{if .Returns}}
					resp.Set(s.{{.Name}}({{if .HasContext}}ctx, {{end}} {{ range .Args }}args.{{.CapitalName}}, {{ end }}))
				{{else}}
					s.{{.Name}}({{if .HasContext}}ctx, {{end}} {{ range .Args }}args.{{.CapitalName}}, {{ end }})
				{{end}}
		{{- end }}
		default:
			resp = zenrpc.NewResponseError(nil, zenrpc.MethodNotFound, "", nil)
		}

		return resp
	}
{{- end }}
`))
)
