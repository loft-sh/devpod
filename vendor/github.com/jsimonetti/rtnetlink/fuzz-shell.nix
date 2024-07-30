with import <nixpkgs> { };
pkgs.mkShell {
  name = "go-fuzz";
  buildInputs = [ go ];
  shellHook = ''
    echo "Fuzz with commands:"
    echo ""
    echo "go test -fuzz=AddressMessage - will start fuzzing Address Messages"
    echo "go test -fuzz=LinkMessage    - will start fuzzing Link Messages"
    echo "go test -fuzz=NeighMessage   - will start fuzzing Neigh Messages"
    echo "go test -fuzz=RouteMessage   - will start fuzzing Route Messages"
    echo "go test -fuzz=RuleMessage    - will start fuzzing Rule Messages"
    echo ""
  '';
}
