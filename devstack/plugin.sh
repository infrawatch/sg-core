function preinstall_sg-core {
	install_package $SG_CORE_CONTAINER_EXECUTABLE
}

function install_sg-core {
	$SG_CORE_CONTAINER_EXECUTABLE pull $SG_CORE_CONTAINER_IMAGE
}

function configure_sg-core {
	sudo mkdir -p `dirname $SG_CORE_CONF`
	sudo cp $SG_CORE_DIR/devstack/sg-core.conf.yaml $SG_CORE_CONF
}

function init_sg-core {
	$SG_CORE_CONTAINER_EXECUTABLE run -v $SG_CORE_CONF:/etc/sg-core.conf.yaml --network host --name sg-core -d $SG_CORE_CONTAINER_IMAGE
}

# check for service enabled
if is_service_enabled sg-core; then

    if [[ "$1" == "stack" && "$2" == "pre-install" ]]; then
        # Set up system services
        echo_summary "Configuring system services sg-core"
        preinstall_sg-core

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
		$SG_CORE_CONTAINER_EXECUTABLE rm sg-core
    fi

    if [[ "$1" == "clean" ]]; then
		$SG_CORE_CONTAINER_EXECUTABLE rmi sg-core:latest
    fi
fi

