// Copyright 2021 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultTimeout = 5 * time.Minute
)

func main() {
	args, err := parseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	event, err := parseGitHubEvent()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	result, err := getPlanPreview(
		ctx,
		event.RepoRemote,
		event.BaseBranch,
		event.HeadBranch,
		event.HeadCommit,
		args.Address,
		args.APIKey,
		args.Timeout,
	)
	if err != nil {
		log.Fatal(err)
	}

	body := makeCommentBody(result)

	comment, err := sendComment(ctx, args.Token, event.PRNumber, body)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Successfully commented plan-preview result on pull request\n%s\n", *comment.HTMLURL)
}

type arguments struct {
	Address string
	APIKey  string
	Token   string
	Timeout time.Duration
}

func parseArgs(args []string) (arguments, error) {
	var out arguments

	for _, arg := range args {
		ps := strings.SplitN(arg, "=", 2)
		if len(ps) != 2 {
			continue
		}
		switch ps[0] {
		case "address":
			out.Address = ps[1]
		case "api-key":
			out.APIKey = ps[1]
		case "token":
			out.Token = ps[1]
		case "timeout":
			d, err := time.ParseDuration(ps[1])
			if err != nil {
				return arguments{}, err
			}
			out.Timeout = d
		}
	}

	if out.Address == "" {
		return out, fmt.Errorf("missing address argument")
	}
	if out.APIKey == "" {
		return out, fmt.Errorf("missing api-key argument")
	}
	if out.Token == "" {
		return out, fmt.Errorf("missing token argument")
	}
	if out.Timeout == 0 {
		out.Timeout = defaultTimeout
	}

	return out, nil
}

func makeCommentBody(r *PlanPreviewResult) string {
	var b strings.Builder
	b.WriteString("RESPONSE")
	b.WriteString(fmt.Sprintf("%v\n", r))

	return b.String()
}
