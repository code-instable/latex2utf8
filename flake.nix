{
  description = "A simple tool to convert LaTeX character identifiers to UTF-8 characters";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }:
    let
      # repo = "code-instable/latex2utf8";
      github_repo_name = "latex2utf8"; # the name that will be used by the binary
      package_name = "lutf";  # Define the package name variable
    in
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages = {
          lutf = pkgs.buildGoModule rec {
            pname = package_name;
            src = ./.;
            version = "1.2.0";
            # ⓘ get the package Hash when first building it
            vendorHash = nixpkgs.lib.fakeHash;
            # vendorHash = "sha256-hocnLCzWN8srQcO3BMNkd2lt0m54Qe7sqAhUxVZlz1k="; 
            buildInputs = [ ];
            output = package_name;

            # rename executable or make a symlink with different name
            postInstall = ''
              # ➀ double name install (github package name + shortcut|alias)
              # ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
              # ln -s $out/bin/${github_repo_name} $out/bin/${package_name} # make a symlink with a shortened desired package name
              # 
              # ➁ single name install
              # ━━━━━━━━━━━━━━━━━━━━━
              mv $out/bin/${github_repo_name} $out/bin/${package_name}
            '';

            meta = with pkgs.lib; {
              description = "A simple CLI tool to convert LaTeX character identifiers to UTF-8 characters";
              homepage = "https://github.com/code-instable/latex2utf8";
              license = licenses.gpl3;
              maintainers = with maintainers; [ code-instable ];
            };
          };

          default = self.packages.${system}.lutf;
        };

        devShells = {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.gotools
            ];
          };
        };

        meta = {
          description = "A simple CLI tool to convert LaTeX character identifiers to UTF-8 characters";
          longDescription = ''
            A simple CLI tool to convert LaTeX character identifiers to UTF-8 characters; its main purpose was to be used in terminal editors such as helix or nvim, inspired by the vscode extension github:ojsheikh/unicode-latex.
          '';
          homepage = "https://github.com/code-instable/latex2utf8";
          mainProgram = "lutf";
        };
      }
    );
}

