package dockerfile

import (
	_ "embed"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

//go:embed test_Dockerfile
var testDockerFileContents string

func TestBuildContextFiles(t *testing.T) {
	dockerFile, err := Parse(testDockerFileContents)
	assert.NilError(t, err)

	fmt.Print(dockerFile.Stages)

	files := dockerFile.BuildContextFiles()
	assert.Equal(t, len(files), 2)
	assert.Equal(t, files[0], "app")
	assert.Equal(t, files[1], "files")
}
