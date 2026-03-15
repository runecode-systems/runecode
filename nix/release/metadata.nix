let
  base = {
    packageName = "runecode";
    version = "0.1.0-alpha.1";

    binaries = [
      "runecode-auditd"
      "runecode-broker"
      "runecode-launcher"
      "runecode-secretsd"
      "runecode-tui"
    ];

    targets = [
      {
        goos = "linux";
        goarch = "amd64";
        archive = "tar.gz";
      }
      {
        goos = "linux";
        goarch = "arm64";
        archive = "tar.gz";
      }
      {
        goos = "darwin";
        goarch = "amd64";
        archive = "tar.gz";
      }
      {
        goos = "darwin";
        goarch = "arm64";
        archive = "tar.gz";
      }
      {
        goos = "windows";
        goarch = "amd64";
        archive = "zip";
      }
      {
        goos = "windows";
        goarch = "arm64";
        archive = "zip";
      }
    ];
  };
in
base
// {
  tag = "v${base.version}";
}
