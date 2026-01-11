cask "{{ .Name }}" do
  desc "{{ .Description }}"
  homepage "{{ .Homepage }}"
  version "{{ .Version }}"

  livecheck do
    skip "Auto-generated on release."
  end

  binary "{{ .Name }}"

  on_macos do
    on_intel do
      url "{{ .DarwinURL }}"
      sha256 "{{ .DarwinSha256 }}"
    end
  end

  on_linux do
    on_intel do
      url "{{ .LinuxURL }}"
      sha256 "{{ .LinuxSha256 }}"
    end
  end

  postflight do
    if OS.mac?
      if system_command("/usr/bin/xattr", args: ["-h"]).exit_status == 0
        system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/{{ .Name }}"]
      end
    end
  end
end
