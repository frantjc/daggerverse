"AppBuild"
{
	"AppID" "{{ .AppID }}"
	"Desc" "{{ .Desc }}"
{{- if .Branch }}
	"SetLive" "{{ .Branch }}"
{{- end }}
	"ContentRoot" "{{ .ContentRoot }}"
	"BuildOutput" "{{ .BuildOutput }}"
	"Depots"
	{
{{- range .Depots }}
		"{{ .DepotID }}"
		{
			"FileMapping"
			{
				"LocalPath" "{{ .Path }}"
				"DepotPath" "."
				"recursive" "{{ if .Recursive }}1{{ else }}0{{ end }}"
			}
		}
{{- end }}
	}
}
