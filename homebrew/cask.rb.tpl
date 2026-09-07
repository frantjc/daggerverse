cask "{{ .Name }}" do
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"
  binary "{{ .Name }}"

  livecheck do
    skip "Auto-generated on release."
  end

  {{- range $goos, $archMap := .OsArch }}
  on_{{ $goos }} do
    {{- range $goarch, $osArchData := $archMap }}
    on_{{ $goarch }} do
      url "{{ $osArchData.URL }}"
      sha256 "{{ $osArchData.Sha256 }}"
    end
    {{- end }}
  end

  {{- end }}
  postflight do
    if OS.mac?
      if system_command("/usr/bin/xattr", args: ["-h"]).exit_status == 0
        system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/{{ .Name }}"]
      end
    end
  end
end
