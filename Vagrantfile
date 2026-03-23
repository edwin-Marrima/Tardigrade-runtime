Vagrant.configure("2") do |config|
  config.vm.box = "hashicorp-education/ubuntu-24-04"
  config.vm.box_version = "0.1.0"

  config.vm.network "forwarded_port", guest: 8080, host: 8080
  config.vm.network "forwarded_port", guest: 8081, host: 8081

  # Install golang
  config.vm.provision "shell", name: "install-dependencies", path: "vagrant-install-dependencies.sh"
  config.vm.provision "shell", name: "install-docker", inline: <<-SHELL
        apt-get update

        # Install required packages
        apt-get install -y ca-certificates curl gnupg git

        # Add Docker's official GPG key
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        chmod a+r /etc/apt/keyrings/docker.gpg

        # Add Docker repository
        echo \
          "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
          "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
          tee /etc/apt/sources.list.d/docker.list > /dev/null

        # Install Docker packages
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  SHELL

  config.vm.synced_folder "./", "/home/vagrant/tardigrade-runtime", type: "rsync",
    rsync__exclude: [".git/", ".DS_Store", "vendor/"]

end

