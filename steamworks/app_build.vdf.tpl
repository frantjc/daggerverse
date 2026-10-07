"AppBuild"
{
	"AppID" "{{ .AppID }}"
	"Desc" "{{ .Desc }}"
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
