@test "generating and linting examples" {
    for d in $(find ../examples/ -maxdepth 1 -mindepth 1 -type d); do
        pushd $d
        rm -rf chart
        go run ../../cmd/katenary convert -f
        helm lint chart
        popd
    done
}
