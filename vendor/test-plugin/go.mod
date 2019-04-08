module github.com/sylabs/singularity

go 1.11

require (
	github.com/containerd/cgroups v0.0.0-20181208203134-65ce98b3dfeb
	github.com/containernetworking/cni v0.6.0
	github.com/containernetworking/plugins v0.0.0-20180606151004-2b8b1ac0af45
	github.com/containers/image v0.0.0-20180612162315-2e4f799f5eba
	github.com/globalsign/mgo v0.0.0-20180615134936-113d3961e731
	github.com/gorilla/websocket v1.2.0
	github.com/kr/pty v1.1.3
	github.com/kubernetes-sigs/cri-o v0.0.0-20180917213123-8afc34092907
	github.com/magiconair/properties v1.8.0
	github.com/opencontainers/image-spec v0.0.0-20180411145040-e562b0440392
	github.com/opencontainers/image-tools v0.0.0-00010101000000-000000000000
	github.com/opencontainers/runtime-spec v0.0.0-20180913141938-5806c3563733
	github.com/opencontainers/runtime-tools v0.6.0
	github.com/opencontainers/selinux v1.0.0-rc1
	github.com/pelletier/go-toml v1.2.0
	github.com/pkg/errors v0.8.0
	github.com/satori/go.uuid v1.2.0
	github.com/seccomp/libseccomp-golang v0.9.0
	github.com/spf13/cobra v0.0.0-20190321000552-67fc4837d267
	github.com/spf13/pflag v1.0.3
	github.com/sylabs/json-resp v0.5.0
	github.com/sylabs/scs-key-client v0.2.0
	github.com/sylabs/sif v1.0.3
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9
	golang.org/x/sys v0.0.0-20190222072716-a9d3bda3a223
	gopkg.in/cheggaaa/pb.v1 v1.0.25
	gopkg.in/yaml.v2 v2.2.2
	test-plugin v0.0.0 // indirect
)

replace (
	github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.0.5
	github.com/opencontainers/image-tools => github.com/sylabs/image-tools v0.0.0-20181006203805-2814f4980568
	golang.org/x/crypto => github.com/sylabs/golang-x-crypto v0.0.0-20181006204705-4bce89e8e9a9
	test-plugin => ../test-plugin
)
