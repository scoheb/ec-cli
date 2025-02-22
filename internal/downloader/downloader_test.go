// Copyright The Enterprise Contract Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

//go:build unit

package downloader

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/enterprise-contract/go-gather/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDownloader struct {
	mock.Mock
}

func (m *mockDownloader) Download(ctx context.Context, dest string, sourceUrls []string) error {
	args := m.Called(ctx, dest, sourceUrls)

	return args.Error(0)
}

func TestDownloader_Download(t *testing.T) {
	tests := []struct {
		name        string
		dest        string
		source      string
		errExpected bool
		err         error
		useGoGather bool
	}{
		{
			name:   "Downloads",
			dest:   "dir",
			source: "https://example.com/org/repo.git",
		},
		{
			name:        "Downloads with go-gather",
			dest:        "dir",
			source:      "https://example.com/org/repo.git",
			useGoGather: true,
		},
		{
			name:        "Fails to download with go-gather",
			dest:        "dir",
			source:      "https://example.com/org/repo.git",
			errExpected: true,
			err:         errors.New("expected error with go-gather"),
			useGoGather: true,
		},
		{
			name:        "Fails to download",
			dest:        "dir",
			source:      "https://example.com/org/repo.git",
			errExpected: true,
			err:         errors.New("expected error"),
		},
		{
			name:        "insecure download",
			dest:        "dir",
			source:      "http://example.com/org/repo.git",
			errExpected: true,
			err:         errors.New("attempting to download from insecure source: http://example.com/org/repo.git"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mockDownloader{}
			ctx := WithDownloadImpl(context.TODO(), &d)

			originalGatherFunction := gatherFunc
			defer func() {
				gatherFunc = originalGatherFunction
			}()

			if tt.useGoGather {
				t.Setenv("USEGOGATHER", "1")
				gatherFunc = func(_ context.Context, _ string, _ string) (metadata.Metadata, error) {
					return nil, tt.err
				}
			} else {
				os.Unsetenv("USEGOGATHER")
				d.On("Download", ctx, tt.dest, []string{tt.source}).Return(tt.err)
			}

			_, err := Download(ctx, tt.dest, tt.source, false)

			if tt.errExpected {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				mock.AssertExpectationsForObjects(t, &d)
			}
		})
	}
}

func TestIsSecure(t *testing.T) {
	secure := []string{
		"./foo",
		"github.com/mitchellh/vagrant",
		"gitlab.com/inkscape/inkscape",
		"bitbucket.org/mitchellh/vagrant",
		"git::https://github.com/mitchellh/vagrant.git",
		"git::ssh://git@example.com/foo/bar",
		"git::git@example.com/foo/bar",
		"https://Aladdin:OpenSesame@www.example.com/index.html", // gitleaks:allow
		"s3::https://s3.amazonaws.com/bucket/foo",
		"s3::https://s3-eu-west-1.amazonaws.com/bucket/foo",
		"bucket.s3.amazonaws.com/foo",
		"bucket.s3-eu-west-1.amazonaws.com/foo/bar",
		"gcs::https://www.googleapis.com/storage/v1/bucket",
		"gcs::https://www.googleapis.com/storage/v1/bucket/foo.zip",
		"www.googleapis.com/storage/v1/bucket/foo",
		"oci::registry.io/repository/image:tag",
	}

	for _, u := range secure {
		assert.True(t, isSecure(u), `Expecting isSecure("%s") = true, but it was false`, u)
	}

	insecure := []string{
		"http://example.com",
		"git::http://github.com/org/repository",
		"hg::http://github.com/org/repository",
		"http::http://github.com/org/repository",
		"s3::http://127.0.0.1:9000/test-bucket/hello.txt?aws_access_key_id=KEYID&aws_access_key_secret=SECRETKEY&region=us-east-2",
	}

	for _, u := range insecure {
		assert.False(t, isSecure(u), `Expecting isSecure("%s") = false, but it was true`, u)
	}
}
