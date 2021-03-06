/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package inventory_test

import (
	"fmt"
	"testing"

	reg "sigs.k8s.io/k8s-container-image-promoter/lib/dockerregistry"
)

func TestImageRemovalCheck(t *testing.T) {
	srcRegName := reg.RegistryName("gcr.io/foo")
	srcRegName2 := reg.RegistryName("gcr.io/foo2")
	destRegName := reg.RegistryName("gcr.io/bar")
	destRC := reg.RegistryContext{
		Name:           destRegName,
		ServiceAccount: "robot",
	}
	srcRC := reg.RegistryContext{
		Name:           srcRegName,
		ServiceAccount: "robot",
		Src:            true,
	}
	srcRC2 := reg.RegistryContext{
		Name:           srcRegName2,
		ServiceAccount: "robot",
		Src:            true,
	}
	registries := []reg.RegistryContext{destRC, srcRC}
	registries2 := []reg.RegistryContext{destRC, srcRC, srcRC2}

	imageA := reg.Image{
		ImageName: "a",
		Dmap: reg.DigestTags{
			"sha256:000": {"0.9"}}}
	imageA2 := reg.Image{
		ImageName: "a",
		Dmap: reg.DigestTags{
			"sha256:111": {"0.9"}}}
	imageB := reg.Image{
		ImageName: "b",
		Dmap: reg.DigestTags{
			"sha256:000": {"0.9"}}}

	var tests = []struct {
		name            string
		check           reg.ImageRemovalCheck
		masterManifests []reg.Manifest
		pullManifests   []reg.Manifest
		expected        error
	}{
		{
			"Empty manifests",
			reg.ImageRemovalCheck{},
			[]reg.Manifest{},
			[]reg.Manifest{},
			nil,
		},
		{
			"Same manifests",
			reg.ImageRemovalCheck{},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC},
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC},
			},
			nil,
		},
		{
			"Different manifests",
			reg.ImageRemovalCheck{},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC},
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageB,
					},
					SrcRegistry: &srcRC},
			},
			fmt.Errorf("The following images were removed in this pull " +
				"request: a"),
		},
		{
			"Promoting same image from different registry",
			reg.ImageRemovalCheck{},
			[]reg.Manifest{
				{
					Registries: registries2,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC},
			},
			[]reg.Manifest{
				{
					Registries: registries2,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC2},
			},
			nil,
		},
		{
			"Promoting image with same name and different digest",
			reg.ImageRemovalCheck{},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageA,
					},
					SrcRegistry: &srcRC},
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						imageA2,
					},
					SrcRegistry: &srcRC},
			},
			fmt.Errorf("The following images were removed in this pull " +
				"request: a"),
		},
	}

	for _, test := range tests {
		masterEdges, _ := reg.ToPromotionEdges(test.masterManifests)
		pullEdges, _ := reg.ToPromotionEdges(test.pullManifests)
		got := test.check.Compare(masterEdges, pullEdges)
		err := checkEqual(got, test.expected)
		checkError(t, err,
			fmt.Sprintf("checkError: test: %v imageRemovalCheck\n", test.name))
	}
}

func TestImageSizeCheck(t *testing.T) {
	srcRegName := reg.RegistryName("gcr.io/foo")
	destRegName := reg.RegistryName("gcr.io/bar")
	destRC := reg.RegistryContext{
		Name:           destRegName,
		ServiceAccount: "robot",
	}
	srcRC := reg.RegistryContext{
		Name:           srcRegName,
		ServiceAccount: "robot",
		Src:            true,
	}
	registries := []reg.RegistryContext{destRC, srcRC}

	image1 := reg.Image{
		ImageName: "foo",
		Dmap: reg.DigestTags{
			"sha256:000": {"0.9"}}}
	image2 := reg.Image{
		ImageName: "bar",
		Dmap: reg.DigestTags{
			"sha256:111": {"0.9"}}}

	var tests = []struct {
		name       string
		check      reg.ImageSizeCheck
		manifests  []reg.Manifest
		imageSizes map[reg.Digest]int
		expected   error
	}{
		{
			"Image size under the max size",
			reg.ImageSizeCheck{
				MaxImageSize:    1,
				DigestImageSize: make(reg.DigestImageSize),
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						image1,
					},
					SrcRegistry: &srcRC},
			},
			map[reg.Digest]int{
				"sha256:000": reg.MBToBytes(1),
			},
			nil,
		},
		{
			"Image size over the max size",
			reg.ImageSizeCheck{
				MaxImageSize:    1,
				DigestImageSize: make(reg.DigestImageSize),
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						image1,
					},
					SrcRegistry: &srcRC},
			},
			map[reg.Digest]int{
				"sha256:000": reg.MBToBytes(5),
			},
			reg.ImageSizeError{
				1,
				map[string]int{
					"foo": reg.MBToBytes(5),
				},
				map[string]int{},
			},
		},
		{
			"Multiple images over the max size",
			reg.ImageSizeCheck{
				MaxImageSize:    1,
				DigestImageSize: make(reg.DigestImageSize),
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						image1,
						image2,
					},
					SrcRegistry: &srcRC},
			},
			map[reg.Digest]int{
				"sha256:000": reg.MBToBytes(5),
				"sha256:111": reg.MBToBytes(10),
			},
			reg.ImageSizeError{
				1,
				map[string]int{
					"foo": reg.MBToBytes(5),
					"bar": reg.MBToBytes(10),
				},
				map[string]int{},
			},
		},
		{
			"Image sizes are <= 0",
			reg.ImageSizeCheck{
				MaxImageSize:    1,
				DigestImageSize: make(reg.DigestImageSize),
			},
			[]reg.Manifest{
				{
					Registries: registries,
					Images: []reg.Image{
						image1,
						image2,
					},
					SrcRegistry: &srcRC},
			},
			map[reg.Digest]int{
				"sha256:000": 0,
				"sha256:111": reg.MBToBytes(-5),
			},
			reg.ImageSizeError{
				1,
				map[string]int{},
				map[string]int{
					"foo": 0,
					"bar": reg.MBToBytes(-5),
				},
			},
		},
	}

	for _, test := range tests {
		test.check.PullEdges, _ = reg.ToPromotionEdges(test.manifests)
		err := checkEqual(len(test.check.PullEdges), len(test.imageSizes))
		checkError(t, err,
			fmt.Sprintf("checkError: test: %v Number of image sizes should be "+
				"equal to the number of edges (ImageSizeCheck)\n", test.name))
		for edge := range test.check.PullEdges {
			test.check.DigestImageSize[edge.Digest] =
				test.imageSizes[edge.Digest]
		}
		got := test.check.Run()
		err = checkEqual(got, test.expected)
		checkError(t, err,
			fmt.Sprintf("checkError: test: %v (ImageSizeCheck)\n", test.name))
	}
}
