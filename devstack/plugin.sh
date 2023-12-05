function install_container_executable {
	if install_package podman; then
		SG_CORE_CONTAINER_EXECUTABLE="podman"
	elif install_package docker.io; then
		sudo chown stack:docker /var/run/docker.sock
		sudo usermod -aG docker stack
		SG_CORE_CONTAINER_EXECUTABLE="docker"
	else
		echo_summary "Couldn't install podman or docker"
		return 1
	fi
	if is_ubuntu; then
		install_package uidmap
	fi
}

### sg-core ###
function install_sg-core {
	$SG_CORE_CONTAINER_EXECUTABLE pull $SG_CORE_CONTAINER_IMAGE
}

function configure_sg-core {
	sudo mkdir -p `dirname $SG_CORE_CONF`
	sudo cp $SG_CORE_DIR/devstack/sg-core-files/sg-core.conf.yaml $SG_CORE_CONF
}

function init_sg-core {
	$SG_CORE_CONTAINER_EXECUTABLE run -v $SG_CORE_CONF:/etc/sg-core.conf.yaml --network host --name sg-core -d $SG_CORE_CONTAINER_IMAGE
}

### prometheus ###
function install_prometheus {
	$SG_CORE_CONTAINER_EXECUTABLE pull $PROMETHEUS_CONTAINER_IMAGE
}

function configure_prometheus {
	BASE_CONFIG_FILE=$SG_CORE_DIR/devstack/prometheus-files/prometheus.yml
	RESULT_CONFIG_FILE=$SG_CORE_WORKDIR/prometheus.yml

	cat $BASE_CONFIG_FILE > $RESULT_CONFIG_FILE

	SERVICES=$(echo $PROMETHEUS_SERVICE_SCRAPE_TARGETS | tr "," "\n")
	for SERVICE in ${SERVICES[@]}
	do
		cat $SG_CORE_DIR/devstack/prometheus-files/scrape_configs/$SERVICE >> $RESULT_CONFIG_FILE
	done

	if [[ $PROMETHEUS_CUSTOM_SCRAPE_TARGETS != "" ]]; then
		echo "  - job_name: 'custom'" >> $RESULT_CONFIG_FILE
		echo "    static_configs:" >> $RESULT_CONFIG_FILE
		echo "      - targets: [$PROMETHEUS_CUSTOM_SCRAPE_TARGETS]" >> $RESULT_CONFIG_FILE
	fi

	sudo mkdir -p `dirname $PROMETHEUS_CONF`
	sudo cp $RESULT_CONFIG_FILE $PROMETHEUS_CONF

	sudo cp $SG_CORE_DIR/devstack/observabilityclient-files/prometheus.yaml /etc/openstack/prometheus.yaml
}

function init_prometheus {
	$SG_CORE_CONTAINER_EXECUTABLE run -v $PROMETHEUS_CONF:/etc/prometheus/prometheus.yml --network host --name prometheus -d $PROMETHEUS_CONTAINER_IMAGE --config.file=/etc/prometheus/prometheus.yml --web.enable-admin-api
}


# check for service enabled
if is_service_enabled sg-core; then

	mkdir $SG_CORE_WORKDIR
	if [[ $SG_CORE_ENABLE = true ]]; then
		if [[ "$1" == "stack" && "$2" == "pre-install" ]]; then
			# Set up system services
			echo_summary "Configuring system services for sg-core"
			install_container_executable

		elif [[ "$1" == "stack" && "$2" == "install" ]]; then
			# Perform installation of service source
			echo_summary "Installing sg-core"
			install_sg-core

		elif [[ "$1" == "stack" && "$2" == "post-config" ]]; then
			# Configure after the other layer 1 and 2 services have been configured
			echo_summary "Configuring sg-core"
			configure_sg-core

		elif [[ "$1" == "stack" && "$2" == "extra" ]]; then
			# Initialize and start the sg-core service
			echo_summary "Initializing sg-core"
			init_sg-core
		fi

		if [[ "$1" == "unstack" ]]; then
			$SG_CORE_CONTAINER_EXECUTABLE stop sg-core
			$SG_CORE_CONTAINER_EXECUTABLE rm -f sg-core
		fi

		if [[ "$1" == "clean" ]]; then
			$SG_CORE_CONTAINER_EXECUTABLE rmi $SG_CORE_CONTAINER_IMAGE
		fi
	fi
	if [[ $PROMETHEUS_ENABLE = true ]]; then
		    if [[ "$1" == "stack" && "$2" == "pre-install" ]]; then
			# Set up system services
			echo_summary "Configuring system services prometheus"
			install_container_executable

		elif [[ "$1" == "stack" && "$2" == "install" ]]; then
			# Perform installation of service source
			echo_summary "Installing prometheus"
			install_prometheus

		elif [[ "$1" == "stack" && "$2" == "post-config" ]]; then
			# Configure after the other layer 1 and 2 services have been configured
			echo_summary "Configuring prometheus"
			configure_prometheus

		elif [[ "$1" == "stack" && "$2" == "extra" ]]; then
			# Initialize and start the prometheus service
			echo_summary "Initializing prometheus"
			init_prometheus
		fi

		if [[ "$1" == "unstack" ]]; then
			$PROMETHEUS_CONTAINER_EXECUTABLE stop prometheus
			$PROMETHEUS_CONTAINER_EXECUTABLE rm -f prometheus
		fi

		if [[ "$1" == "clean" ]]; then
			$PROMETHEUS_CONTAINER_EXECUTABLE rmi $PROMETHEUS_CONTAINER_IMAGE
		fi

	fi
	rm -rf $SG_CORE_WORKDIR
fi

