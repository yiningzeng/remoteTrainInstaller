package main

import (
	"bufio"
	"fmt"
	//"github.com/briandowns/spinner"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	//"time"
)

const nvidiaDriverAndCuda = "#wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/cuda-ubuntu1804.pin\n" +
	"sudo mv local-software/cuda-ubuntu1804.pin /etc/apt/preferences.d/cuda-repository-pin-600\n" +
	"#wget http://developer.download.nvidia.com/compute/cuda/11.0.1/local_installers/cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb\n" +
	"sudo dpkg -i local-software/cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb\n" +
	"sudo apt-key add /var/cuda-repo-ubuntu1804-11-0-local/7fa2af80.pub\n" +
	"sudo apt-get update\n" +
	"sudo apt-get -y install cuda"
const dockerInstaller = "sudo apt-get remove docker docker-engine docker.io containerd runc\n" +
	"sudo apt-get install apt-transport-https ca-certificates curl gnupg-agent software-properties-common\n" +
	"curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -\n" +
	"sudo apt-key fingerprint 0EBFCD88\n" +
	"sudo add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"\n" +
	"sudo apt-get update\n" +
	"sudo apt-get -y install docker-ce docker-ce-cli containerd.io\n" +
	"sudo mkdir -p /etc/docker\n" +
	"sudo tee /etc/docker/daemon.json <<-'EOF'\n{\n\"registry-mirrors\": [\"https://1b0x9gyw.mirror.aliyuncs.com\"]\n}\nEOF\n" +
	"sudo systemctl daemon-reload\n" +
	"sudo systemctl restart docker\n"

const createNet = "sudo docker network create --subnet=10.10.0.0/16 ai\n"
const webService = "sudo  docker run \\\n" +
	"            --name service-web \\\n" +
	"            -p 80:80 \\\n" +
	"            -v /opt/remote_train_web/:/usr/local/apache2/htdocs/ \\\n" +
	"            --net ai --ip 10.10.0.5 \\\n" +
	"            --add-host service-postgresql:10.10.0.4 \\\n" +
	"            --add-host service-rabbitmq:10.10.0.3 \\\n" +
	"            --add-host service-ftp:10.10.0.2 \\\n" +
	"            --add-host service-web:10.10.0.5 \\\n" +
	"            --restart=always \\\n" +
	"            -d registry.cn-hangzhou.aliyuncs.com/baymin/remote-train:web-v3.5"

const postgresqlService = "sudo docker run --name postgresql \\\n" +
	"            -e POSTGRES_PASSWORD=baymin1024 \\\n" +
	"            -p 5432:5432 \\\n" +
	"            --net ai --ip 10.10.0.4 \\\n" +
	"            --add-host service-postgresql:10.10.0.4 \\\n" +
	"            --add-host service-rabbitmq:10.10.0.3 \\\n" +
	"            --add-host service-ftp:10.10.0.2 \\\n" +
	"            --add-host service-web:10.10.0.5 \\\n" +
	"            --restart=always \\\n" +
	"			 -d postgres:9.6"
const rabbitmqService = "sudo docker run --name service-rabbitmq \\\n" +
	"            -e RABBITMQ_DEFAULT_USER=baymin \\\n" +
	"            -e RABBITMQ_DEFAULT_PASS=baymin1024 \\\n" +
	"            --net ai --ip 10.10.0.3 \\\n" +
	"            -p 5672:5672 \\\n" +
	"            -p 15672:15672 \\\n" +
	"            --add-host service-postgresql:10.10.0.4 \\\n" +
	"            --add-host service-rabbitmq:10.10.0.3 \\\n" +
	"            --add-host service-ftp:10.10.0.2 \\\n" +
	"            --add-host service-web:10.10.0.5 \\\n" +
	"            --restart=always \\\n" +
	"            -d registry.cn-hangzhou.aliyuncs.com/baymin/remote-train:rabbitmq"
const ftpService = "sudo docker run -d \\\n" +
	"            --name service-ftp \\\n" +
	"            -v /assets/:/home/vsftpd \\\n" +
	"            -p 20:20 -p 21:21 -p 47400-47470:47400-47470 \\\n" +
	"            -e FTP_USER=baymin \\\n" +
	"            -e FTP_PASS=baymin1024 \\\n" +
	"            -e PASV_ADDRESS=192.168.31.157 \\\n" +
	"            --net ai \\\n" +
	"            --ip 10.10.0.2 \\\n" +
	"            --add-host service-postgresql:10.10.0.4 \\\n" +
	"            --add-host service-rabbitmq:10.10.0.3 \\\n" +
	"            --add-host service-ftp:10.10.0.2 \\\n" +
	"            --add-host service-web:10.10.0.5 \\\n" +
	"            --restart=always \\\n" +
	"            registry.cn-hangzhou.aliyuncs.com/baymin/remote-train:ftp"
const createInstallNvidiaDockerSh = "sudo tee /var/local/install-nvidia-docker.sh <<-'EOF'\n" +
	"#!/bin/bash\n" +
	"# Add the package repositories\n" +
	"distribution=$(. /etc/os-release;echo $ID$VERSION_ID)\n" +
	"curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -\n" +
	"curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list\n" +
	"sudo apt-get update && sudo apt-get install -y nvidia-container-toolkit\n" +
	"sudo systemctl restart docker\n" +
	"EOF"
const createInstallNvidiaDocker = "sudo tee /lib/systemd/system/installNvidiaDocker.service <<-'EOF'\n" +
	"[Unit]\n" +
	"Description=nvidia-docker-install Service\n" +
	"After=network.target\n" +
	"\n" +
	"[Service]\n" +
	"Type=simple\n" +
	"User=nobody\n" +
	"Restart=on-failure\n" +
	"RestartSec=5s\n" +
	"ExecStart=/var/local/install-nvidia-docker.sh\n" +
	"ExecReload=/var/local/install-nvidia-docker.sh\n" +
	"\n" +
	"[Install]\n" +
	"WantedBy=multi-user.target\n" +
	"EOF"
//var s = spinner.New(spinner.CharSets[35], 100*time.Millisecond)  // Build our new spinner

// 判断文件是否存在
func IsExist(fileAddr string)(bool){
	// 读取文件信息，判断文件是否存在
	_,err := os.Stat(fileAddr)
	if err!=nil{
		log.Println(err)
		if os.IsExist(err){  // 根据错误类型进行判断
			return true
		}
		return false
	}
	return true
}

func execCommand(commandName string, params []string) bool {

	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(commandName, params[:len(params)-2]...)

	//显示运行的命令
	//fmt.Println(cmd.Args)
	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()

	if err != nil {
		fmt.Println(err)
		return false
	}

	cmd.Start()
	if params[len(params) -2 ] != "" {
		fmt.Print( "\n")
		fmt.Printf("%s ", params[len(params) -2 ])      //输出a
		//s.Start()
	} else if !strings.Contains(params[1], "sudo echo starting...")  {
		//s.Start()
	}
	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		  // Append text after the spinner// Start the spinner
		//fmt.Printf("%d\n", i)      //输出a
		//fmt.Println(strings.Replace(line, "\n", "", -1))
		fmt.Println(line)
	}
	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	cmd.Wait()
	//s.Stop()
	return true
}

func getPar(par string, args ...string) []string{
	if len(args) == 1 {
		return []string{"-c", par, args[0], ""}
	} else if len(args) == 2 {
		return []string{"-c", par, args[0], args[1]}
	} else {
		return []string{"-c", par, "", ""}
	}
}

func main() {
 	debug := false
	//_ = execCommand("/bin/bash", getPar("echo 安装之前需要自行安装Anaconda3: \"https://mirrors.tuna.tsinghua.edu.cn/anaconda/archive/Anaconda3-2019.10-Linux-x86_64.sh\"\\\n && echo \"请确认是否已经安装，如果未安装[Ctrl+c]取消安装\"\\\n && sudo echo starting..."))
	//s.Color("red", "bold") // Set the spinner color to a bold red
	if !debug {
		_ = execCommand("/bin/bash", getPar("sudo apt update"))
	}
	if !IsExist("./local-software/Anaconda3-2019.10-Linux-x86_64.sh") {
		fmt.Print("正在下载Anaconda3-2019.10-Linux-x86_64.sh")
		_ = execCommand("/bin/bash", getPar("wget https://mirrors.tuna.tsinghua.edu.cn/anaconda/archive/Anaconda3-2019.10-Linux-x86_64.sh -O local-software/Anaconda3-2019.10-Linux-x86_64.sh"))
	}
	if !IsExist("./local-software/cuda-ubuntu1804.pin") {
		fmt.Print("正在下载cuda-ubuntu1804.pin")
		_ = execCommand("/bin/bash", getPar("wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu1804/x86_64/cuda-ubuntu1804.pin -O local-software/cuda-ubuntu1804.pin"))
	}
	if !IsExist("./local-software/cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb") {
		fmt.Print("正在下载cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb")
		_ = execCommand("/bin/bash", getPar("wget http://developer.download.nvidia.com/compute/cuda/11.0.1/local_installers/cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb -O local-software/cuda-repo-ubuntu1804-11-0-local_11.0.1-450.36.06-1_amd64.deb"))
	}
	if !IsExist("/var/local/install-nvidia-docker.sh") {
		fmt.Print("生成服务")
		_ = execCommand("/bin/bash", getPar(createInstallNvidiaDockerSh))
	}
	if !IsExist("/lib/systemd/system/installNvidiaDocker.service") {
		fmt.Print("生成服务")
		_ = execCommand("/bin/bash", getPar(createInstallNvidiaDocker))
	}
	_ = execCommand("/bin/bash", getPar("sudo chmod +x /var/local/install-nvidia-docker.sh"))
	// 设置开机启动
	_ = execCommand("/bin/bash", getPar("sudo systemctl enable /lib/systemd/system/installNvidiaDocker.service"))

	// 安装anconda
	_ = execCommand("/bin/bash", getPar("sudo sh local-software/Anaconda3-2019.10-Linux-x86_64.sh -b -u"))
	_ = execCommand("/bin/bash", getPar(ftpService, "正在下载和开启FTP上传服务"))

	_ = execCommand("/bin/bash", getPar(nvidiaDriverAndCuda))
	_ = execCommand("/bin/bash", getPar("sudo apt-get -y install curl"))
	_ = execCommand("/bin/bash", getPar("mkdir -p /opt/remote_train_web /opt/remote_train_service /assets"))
	_ = execCommand("/bin/bash", getPar("sudo chmod -R 777 /opt/remote_train_web /opt/remote_train_service /assets"))
	_ = execCommand("/bin/bash", getPar("sudo curl \"http://pan.qtingvision.com:888/s/LCP3rwj2GFJ4RmE/download?path=%2F&files=web.tar.gz\" -o /opt/remote_train_web/web.tar.gz", "正在下载安装服务支持包[1/4]"))

	_ = execCommand("/bin/bash", getPar("sudo tar -xzf /opt/remote_train_web/web.tar.gz -C /opt/remote_train_web"))
	_ = execCommand("/bin/bash", getPar("source ~/.bashrc && pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple && pip install visdom", "正在下载安装服务支持包[2/4]"))

	_ = execCommand("/bin/bash", getPar("sudo curl \"http://pan.qtingvision.com:888/s/LCP3rwj2GFJ4RmE/download?path=%2F&files=visdom-static.tar.gz\" -o /opt/remote_train_web/visdom-static.tar.gz", "正在下载安装服务支持包[3/4]"))
	_ = execCommand("/bin/bash", getPar("sudo tar -xzf /opt/remote_train_web/visdom-static.tar.gz -C ~/anaconda3/lib/python3.7/site-packages/visdom/static"))

	_ = execCommand("/bin/bash", getPar("sudo curl \"http://pan.qtingvision.com:888/s/LCP3rwj2GFJ4RmE/download?path=%2F&files=api.tar.gz\" -o /opt/remote_train_service/api.tar.gz", "正在下载安装服务支持包[4/4]"))
	_ = execCommand("/bin/bash", getPar("sudo tar -xzf /opt/remote_train_service/api.tar.gz -C /opt/remote_train_service && sudo mv -f /opt/remote_train_service/dockertrain /usr/local/bin/dockertrain"))


	if !debug {
		_ = execCommand("/bin/bash", getPar(dockerInstaller))
	}
	_ = execCommand("/bin/bash", getPar(createNet))
	_ = execCommand("/bin/bash", getPar(postgresqlService, "正在下载和开启数据库服务"))
	_ = execCommand("/bin/bash", getPar(rabbitmqService, "正在下载和开启队列服务"))
	_ = execCommand("/bin/bash", getPar(webService, "正在下载和开启后台管理服务"))
	_ = execCommand("/bin/bash", getPar(ftpService, "正在下载和开启FTP上传服务"))
	//cmd := exec.Command("touch", "test_file")
	//err := cmd.Run()
	//cmd = exec.Command("/bin/bash", "-c", "sudo apt-get remove docker docker-engine docker.io containerd runc\n" +
	//	"sudo apt-get update \n" +
	//	"sudo apt-get install apt-transport-https ca-certificates curl gnupg-agent software-properties-common")
	//err = cmd.Run()
	//if err != nil {
	//	fmt.Println("Execute Command failed:" + err.Error())
	//	return
	//}
	//fmt.Print("a", "\n")      //输出a
	//fmt.Print("a", "b", "\n") //输出ab
	//fmt.Print('a', "\n")      //输出97
	//fmt.Print('a', 'b', "\n") //输出97 98   字符之间会输出一个空格
	//fmt.Print(12, "\n")       //输出12
	//fmt.Print(12,13, "\n")   //输出12 13   数值之间输出一个空格
	//fmt.Printf("%v", "asdsds")
	//fmt.Printf("%d\n", 10)
	fmt.Printf("所有服务安装完成")
}