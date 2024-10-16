/*
 *     Copyright 2024 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint
	. "github.com/onsi/gomega"    //nolint
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	schedulerclient "d7y.io/dragonfly/v2/pkg/rpc/scheduler/client"
	"d7y.io/dragonfly/v2/test/e2e/v2/util"
)

var _ = Describe("Clients Leaving", func() {
	Context("normally", func() {
		It("number of hosts should be ok", Label("host", "leave"), func() {
			// create scheduler GRPC client
			grpcCredentials := insecure.NewCredentials()
			schedulerClient, err := schedulerclient.GetV2ByAddr(context.Background(), ":8002", grpc.WithTransportCredentials(grpcCredentials))
			Expect(err).NotTo(HaveOccurred())

			// get host count
			hostCount := util.Servers[util.SeedClientServerName].Replicas + util.Servers[util.ClientServerName].Replicas
			time.Sleep(10 * time.Minute)
			Expect(getHostCountFromScheduler(schedulerClient)).To(Equal(hostCount))

			// get client pod name in master node
			podName, err := util.GetClientPodNameInMaster()
			Expect(err).NotTo(HaveOccurred())

			// taint master node
			out, err := util.KubeCtlCommand("-n", util.DragonflyNamespace, "taint", "nodes", "kind-control-plane", "master:NoSchedule").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			// delete client pod in master, client will leave normally
			out, err = util.KubeCtlCommand("-n", util.DragonflyNamespace, "delete", "pod", podName, "--grace-period=30").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			// wait fot the client to leave gracefully
			time.Sleep(30 * time.Second)
			Expect(getHostCountFromScheduler(schedulerClient)).To(Equal(hostCount - 1))

			// remove taint in master node
			out, err = util.KubeCtlCommand("-n", util.DragonflyNamespace, "taint", "nodes", "kind-control-plane", "master:NoSchedule-").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("abnormally", func() {
		It("number of hosts should be ok", Label("host", "leave"), func() {
			// create scheduler GRPC client
			grpcCredentials := insecure.NewCredentials()
			schedulerClient, err := schedulerclient.GetV2ByAddr(context.Background(), ":8002", grpc.WithTransportCredentials(grpcCredentials))
			Expect(err).NotTo(HaveOccurred())

			// get host count
			hostCount := util.Servers[util.SeedClientServerName].Replicas + util.Servers[util.ClientServerName].Replicas
			time.Sleep(30 * time.Second)
			Expect(getHostCountFromScheduler(schedulerClient)).To(Equal(hostCount))

			// get client pod name in master node
			podName, err := util.GetClientPodNameInMaster()
			Expect(err).NotTo(HaveOccurred())

			// taint master node
			out, err := util.KubeCtlCommand("-n", util.DragonflyNamespace, "taint", "nodes", "kind-control-plane", "master:NoSchedule").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			// force delete client pod in master, client will leave abnormally
			out, err = util.KubeCtlCommand("-n", util.DragonflyNamespace, "delete", "pod", podName, "--force", "--grace-period=0").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())

			// wait for host gc
			time.Sleep(2 * time.Minute)
			Expect(getHostCountFromScheduler(schedulerClient)).To(Equal(hostCount - 1))

			// remove taint in master node
			out, err = util.KubeCtlCommand("-n", util.DragonflyNamespace, "taint", "nodes", "kind-control-plane", "master:NoSchedule-").CombinedOutput()
			fmt.Println(string(out))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getHostCountFromScheduler(schedulerClient schedulerclient.V2) (hostCount int) {
	response, err := schedulerClient.ListHosts(context.Background(), "")
	fmt.Println(response, err)
	Expect(err).NotTo(HaveOccurred())

	hosts := response.Hosts
	for _, host := range hosts {
		// HostID: "10.244.0.13-dragonfly-seed-client-0-seed"
		// PeerHostID: "3dba4916d8271d6b71bb20e95a0b5494c9a941ab7ef3567f805abca8614dc128"
		if strings.Contains(host.Id, "-") {
			hostCount++
		}
	}
	return
}
