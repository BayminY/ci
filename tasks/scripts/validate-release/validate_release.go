package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	var s strings.Builder
	for _, flag := range *i {
		s.WriteString(flag)
	}

	return s.String()
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// 2 required flag args pointing to the files described below
// --semver-file
// --docker-digest
func main() {
	var semverFilePaths, dockerDigestPaths arrayFlags
	flag.Var(&semverFilePaths, "semver-file", "path to file containing semver version")
	flag.Var(&dockerDigestPaths, "docker-digest", "path to file containing docker digest")
	flag.Parse()

	versions := make(map[string]string)
	var releaseVersion string

	for _, semverFilePath := range semverFilePaths {
		semverVersion, err := readFile(semverFilePath)
		if err != nil {
			panic(err)
		}

		fmt.Printf("fetched semver version %s from %s\n", semverVersion, semverFilePath)

		releaseVersion = semverVersion
		versions[semverVersion] = semverFilePath
	}

	if len(versions) > 1 {
		v, err := json.Marshal(versions)
		if err != nil {
			panic(err)
		}

		panic(fmt.Errorf("release versions not unique in %v", string(v)))
	}

	fmt.Printf("release versions validation pass...\n")

	digests := make(map[string]string)
	for _, dockerDigestPath := range dockerDigestPaths {
		digest, err := readFile(dockerDigestPath)
		if err != nil {
			panic(err)
		}

		digests[digest] = dockerDigestPath
	}

	err := ValidateDockerDigests([]string{releaseVersion, releaseVersion + "-ubuntu"}, digests)
	if err != nil {
		panic(err)
	}

	fmt.Printf("docker tags validation pass...\n")
}

func readFile(filePath string) (string, error) {
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(fileContents)), nil
}

func ValidateDockerDigests(tags []string, digestMap map[string]string) error {
	repo, err := name.NewRepository("concourse/concourse", name.WeakValidation)
	if err != nil {
		return err
	}

	tag := new(name.Tag)

	for _, t := range tags {
		*tag = repo.Tag(t)

		image, err := remote.Image(*tag)
		if err != nil {
			return err
		}

		digest, err := image.Digest()
		if err != nil {
			return err
		}

		digestString := digest.String()

		fmt.Printf("fetched digest %s from %s:%s\n", digestString, repo.String(), t)

		if _, ok := digestMap[digestString]; !ok {
			return fmt.Errorf("image with tag %s has mismatch digest %s", t, digestString)
		}

		fmt.Printf("image with tag %s has matched digest in %s\n", t, digestMap[digestString])
	}

	return nil
}
