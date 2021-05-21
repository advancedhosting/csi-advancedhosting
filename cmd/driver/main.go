/*
Copyright 2021 Advanced Hosting

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"

	"github.com/advancedhosting/csi-advancedhosting/driver"
	"k8s.io/klog"
)

func main() {

	var (
		endpoint  = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint.")
		apiUrl    = flag.String("url", "https://api.websa.com", "Advanced Hosting api url.")
		apiToken  = flag.String("token", "", "Advanced Hosting access token.")
		clusterID = flag.String("cluster-id", "", "Cluster ID.")
	)
	klog.InitFlags(nil)
	flag.Parse()

	options := &driver.DriverOptions{
		Endpoint:  *endpoint,
		Url:       *apiUrl,
		Token:     *apiToken,
		ClusterID: *clusterID,
	}
	csiDriver, err := driver.NewDriver(options)
	if err != nil {
		klog.Fatalln(err)
	}

	if err := csiDriver.Run(); err != nil {
		klog.Fatalln(err)
	}

}
