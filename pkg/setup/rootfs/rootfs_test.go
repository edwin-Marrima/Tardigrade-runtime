package rootfs

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testImage = "registry.example.com/tardigrade/rootfs:latest"

// call records a single invocation of run.
type call struct {
	name string
	args []string
}

// mockRun replaces the package-level run variable with a mock that records
// every call. failOn maps a command name to the error it should return;
// all other calls succeed. The returned restore function must be deferred.
func mockRun(t *testing.T, failOn map[string]error) (*[]call, func()) {
	t.Helper()
	calls := &[]call{}
	orig := run
	run = func(name string, args ...string) error {
		*calls = append(*calls, call{name, args})
		if err, ok := failOn[name]; ok {
			return err
		}
		return nil
	}
	return calls, func() { run = orig }
}

// outputFile creates a real temp file whose path is used as outputFilePath.
func outputFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "rootfs-*.img")
	require.NoError(t, err)
	f.Close()
	return f.Name()
}

func commandNames(calls []call) []string {
	names := make([]string, len(calls))
	for i, c := range calls {
		names[i] = c.name
	}
	return names
}

// ---- happy path ---------------------------------------------------------------

func TestSetupRootfs_Success_CommandSequence(t *testing.T) {
	calls, restore := mockRun(t, nil)
	defer restore()

	require.NoError(t, setupRootfs(testImage, outputFile(t)))

	assert.Equal(t,
		[]string{"docker", "docker", "docker", "fallocate", "mkfs.ext4", "mount", "tar"},
		commandNames(*calls),
	)
}

func TestSetupRootfs_Success_PullUsesImage(t *testing.T) {
	calls, restore := mockRun(t, nil)
	defer restore()

	require.NoError(t, setupRootfs(testImage, outputFile(t)))

	pullCall := (*calls)[0]
	assert.Equal(t, "docker", pullCall.name)
	assert.Equal(t, []string{"pull", testImage}, pullCall.args)
}

func TestSetupRootfs_Success_CreateUsesImage(t *testing.T) {
	calls, restore := mockRun(t, nil)
	defer restore()

	require.NoError(t, setupRootfs(testImage, outputFile(t)))

	createCall := (*calls)[1]
	assert.Equal(t, "docker", createCall.name)
	assert.Equal(t, []string{"create", "--name", containerName, testImage}, createCall.args)
}

func TestSetupRootfs_Success_OutputFilePreserved(t *testing.T) {
	_, restore := mockRun(t, nil)
	defer restore()

	out := outputFile(t)
	require.NoError(t, setupRootfs(testImage, out))

	_, err := os.Stat(out)
	assert.NoError(t, err, "output file should survive a successful run")
}

// ---- failure paths ------------------------------------------------------------

func TestSetupRootfs_PullFails(t *testing.T) {
	pullErr := errors.New("pull failed")
	calls, restore := mockRun(t, map[string]error{"docker": pullErr})
	defer restore()

	err := setupRootfs(testImage, outputFile(t))
	require.Error(t, err)
	assert.ErrorIs(t, err, pullErr)

	// Only the pull call; no container created so no cleanup needed.
	assert.Equal(t, []string{"docker"}, commandNames(*calls))
}

func TestSetupRootfs_CreateFails(t *testing.T) {
	createErr := errors.New("create failed")
	dockerCalls := 0
	orig := run
	defer func() { run = orig }()

	var calls []call
	run = func(name string, args ...string) error {
		calls = append(calls, call{name, args})
		if name == "docker" {
			dockerCalls++
			if dockerCalls == 2 { // second docker call is "create"
				return createErr
			}
		}
		return nil
	}

	err := setupRootfs(testImage, outputFile(t))
	require.Error(t, err)
	assert.ErrorIs(t, err, createErr)

	// pull + create; cleaner not yet registered so no cleanup calls.
	assert.Equal(t, []string{"docker", "docker"}, commandNames(calls))
}

func TestSetupRootfs_ExportFails(t *testing.T) {
	exportErr := errors.New("export failed")
	dockerCalls := 0
	orig := run
	defer func() { run = orig }()

	var calls []call
	run = func(name string, args ...string) error {
		calls = append(calls, call{name, args})
		if name == "docker" {
			dockerCalls++
			if dockerCalls == 3 { // third docker call is "export"
				return exportErr
			}
		}
		return nil
	}

	err := setupRootfs(testImage, outputFile(t))
	require.Error(t, err)
	assert.ErrorIs(t, err, exportErr)

	// pull, create, export(fail), cleanup: docker rm
	assert.Equal(t, []string{"docker", "docker", "docker", "docker"}, commandNames(calls))
	last := calls[len(calls)-1]
	assert.Equal(t, []string{"rm", "-f", containerName}, last.args)
}

func TestSetupRootfs_FallocateFails(t *testing.T) {
	fallocErr := errors.New("no space left")
	calls, restore := mockRun(t, map[string]error{"fallocate": fallocErr})
	defer restore()

	err := setupRootfs(testImage, outputFile(t))
	require.Error(t, err)
	assert.ErrorIs(t, err, fallocErr)

	names := commandNames(*calls)
	assert.Contains(t, names, "fallocate")
	assert.NotContains(t, names, "mkfs.ext4")
	// container cleanup must run
	last := (*calls)[len(*calls)-1]
	assert.Equal(t, "docker", last.name)
	assert.Equal(t, []string{"rm", "-f", containerName}, last.args)
}

func TestSetupRootfs_MkfsFails(t *testing.T) {
	mkfsErr := errors.New("mkfs failed")
	calls, restore := mockRun(t, map[string]error{"mkfs.ext4": mkfsErr})
	defer restore()

	out := outputFile(t)
	err := setupRootfs(testImage, out)
	require.Error(t, err)
	assert.ErrorIs(t, err, mkfsErr)

	assert.NotContains(t, commandNames(*calls), "mount")
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "output file should be removed after mkfs failure")
}

func TestSetupRootfs_MountFails(t *testing.T) {
	mountErr := errors.New("mount failed")
	calls, restore := mockRun(t, map[string]error{"mount": mountErr})
	defer restore()

	out := outputFile(t)
	err := setupRootfs(testImage, out)
	require.Error(t, err)
	assert.ErrorIs(t, err, mountErr)

	names := commandNames(*calls)
	assert.NotContains(t, names, "umount")
	assert.NotContains(t, names, "tar")
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "output file should be removed after mount failure")
}

func TestSetupRootfs_TarFails(t *testing.T) {
	tarErr := errors.New("tar failed")
	calls, restore := mockRun(t, map[string]error{"tar": tarErr})
	defer restore()

	out := outputFile(t)
	err := setupRootfs(testImage, out)
	require.Error(t, err)
	assert.ErrorIs(t, err, tarErr)

	names := commandNames(*calls)
	assert.Contains(t, names, "umount")
	_, statErr := os.Stat(out)
	assert.True(t, os.IsNotExist(statErr), "output file should be removed after tar failure")
}

// ---- cleanup ordering --------------------------------------------------------

func TestSetupRootfs_TarFails_UmountBeforeDockerRm(t *testing.T) {
	calls, restore := mockRun(t, map[string]error{"tar": errors.New("tar failed")})
	defer restore()

	require.Error(t, setupRootfs(testImage, outputFile(t)))

	umountIdx, dockerRmIdx := -1, -1
	for i, c := range *calls {
		switch c.name {
		case "umount":
			umountIdx = i
		case "docker":
			if c.args[0] == "rm" {
				dockerRmIdx = i
			}
		}
	}
	require.NotEqual(t, -1, umountIdx, "umount should be called")
	require.NotEqual(t, -1, dockerRmIdx, "docker rm should be called")
	assert.Less(t, umountIdx, dockerRmIdx, "umount must happen before docker rm (LIFO cleanup)")
}
