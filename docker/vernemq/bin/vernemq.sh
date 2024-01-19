#!/usr/bin/env sh

NET_INTERFACE=$(route | grep '^default' | grep -o '[^ ]*$')
NET_INTERFACE=${DOCKER_NET_INTERFACE:-${NET_INTERFACE}}
IP_ADDRESS=$(ip -4 addr show ${NET_INTERFACE} | grep -oE '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}' | sed -e "s/^[[:space:]]*//" | head -n 1)
IP_ADDRESS=${DOCKER_IP_ADDRESS:-${IP_ADDRESS}}

VERNEMQ_ETC_DIR="/vernemq/etc"
VERNEMQ_VM_ARGS_FILE="${VERNEMQ_ETC_DIR}/vm.args"
VERNEMQ_CONF_FILE="${VERNEMQ_ETC_DIR}/vernemq.conf"
VERNEMQ_CONF_LOCAL_FILE="${VERNEMQ_ETC_DIR}/vernemq.conf.local"

SECRETS_KUBERNETES_DIR="/var/run/secrets/kubernetes.io/serviceaccount"

# Function to check istio readiness
istio_health() {
  cmd=$(curl -s http://localhost:15021/healthz/ready > /dev/null)
  status=$?
  return $status
}

# Ensure we have all files and needed directory write permissions
if [ ! -d ${VERNEMQ_ETC_DIR} ]; then
  echo "Configuration directory at ${VERNEMQ_ETC_DIR} does not exist, exiting" >&2
  exit 1
fi
if [ ! -f ${VERNEMQ_VM_ARGS_FILE} ]; then
  echo "ls -l ${VERNEMQ_ETC_DIR}"
  ls -l ${VERNEMQ_ETC_DIR}
  echo "###" >&2
  echo "### Configuration file ${VERNEMQ_VM_ARGS_FILE} does not exist, exiting" >&2
  echo "###" >&2
  exit 1
fi
if [ ! -w ${VERNEMQ_VM_ARGS_FILE} ]; then
  echo "# whoami"
  whoami
  echo "# ls -l ${VERNEMQ_ETC_DIR}"
  ls -l ${VERNEMQ_ETC_DIR}
  echo "###" >&2
  echo "### Configuration file ${VERNEMQ_VM_ARGS_FILE} exists, but there are no write permissions! Exiting." >&2
  echo "###" >&2
  exit 1
fi
if [ ! -s ${VERNEMQ_VM_ARGS_FILE} ]; then
  echo "ls -l ${VERNEMQ_ETC_DIR}"
  ls -l ${VERNEMQ_ETC_DIR}
  echo "###" >&2
  echo "### Configuration file ${VERNEMQ_VM_ARGS_FILE} is empty! This will not work." >&2
  echo "### Exiting now." >&2
  echo "###" >&2
  exit 1
fi

# Ensure the Erlang node name is set correctly
if env | grep "DOCKER_VERNEMQ_NODENAME" -q; then
    sed -i.bak -r "s/-name VerneMQ@.+/-name VerneMQ@${DOCKER_VERNEMQ_NODENAME}/" ${VERNEMQ_VM_ARGS_FILE}
else
    if [ -n "$DOCKER_VERNEMQ_SWARM" ]; then
        NODENAME=$(hostname -i)
        sed -i.bak -r "s/VerneMQ@.+/VerneMQ@${NODENAME}/" ${VERNEMQ_VM_ARGS_FILE}
    else
        sed -i.bak -r "s/-name VerneMQ@.+/-name VerneMQ@${IP_ADDRESS}/" ${VERNEMQ_VM_ARGS_FILE}
    fi
fi

if env | grep "DOCKER_VERNEMQ_DISCOVERY_NODE" -q; then
    discovery_node=$DOCKER_VERNEMQ_DISCOVERY_NODE
    if [ -n "$DOCKER_VERNEMQ_SWARM" ]; then
        tmp=''
        while [[ -z "$tmp" ]]; do
            tmp=$(getent hosts tasks.$discovery_node | awk '{print $1}' | head -n 1)
            sleep 1
        done
        discovery_node=$tmp
    fi
    if [ -n "$DOCKER_VERNEMQ_COMPOSE" ]; then
        tmp=''
        while [[ -z "$tmp" ]]; do
            tmp=$(getent hosts $discovery_node | awk '{print $1}' | head -n 1)
            sleep 1
        done
        discovery_node=$tmp
    fi

    sed -i.bak -r "/-eval.+/d" ${VERNEMQ_VM_ARGS_FILE}
    echo "-eval \"vmq_server_cmd:node_join('VerneMQ@$discovery_node')\"" >> ${VERNEMQ_VM_ARGS_FILE}
fi

# If you encounter "SSL certification error (subject name does not match the host name)", you may try to set DOCKER_VERNEMQ_KUBERNETES_INSECURE to "1".
insecure=""
if env | grep "DOCKER_VERNEMQ_KUBERNETES_INSECURE" -q; then
    echo "Using curl with \"--insecure\" argument to access kubernetes API without matching SSL certificate"
    insecure="--insecure"
fi

if env | grep "DOCKER_VERNEMQ_KUBERNETES_ISTIO_ENABLED" -q; then
    istio_health
    while [ $status != 0 ]; do
        istio_health
        sleep 1
    done
    echo "Istio ready"
fi

# Function to call a HTTP GET request on the given URL Path, using the hostname
# of the current k8s cluster name. Usage: "k8sCurlGet /my/path"
function k8sCurlGet () {
    local urlPath=$1

    local hostname="kubernetes.default.svc.${DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME}"
    local certsFile="${SECRETS_KUBERNETES_DIR}/ca.crt"
    local token=$(cat ${SECRETS_KUBERNETES_DIR}/token)
    local header="Authorization: Bearer ${token}"
    local url="https://${hostname}/${urlPath}"

    curl -sS ${insecure} --cacert ${certsFile} -H "${header}" ${url} \
      || ( echo "### Error on accessing URL ${url}" )
}

DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME=${DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME:-cluster.local}
if [ -d "${SECRETS_KUBERNETES_DIR}" ] ; then
    # Let's get the namespace if it isn't set
    DOCKER_VERNEMQ_KUBERNETES_NAMESPACE=${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE:-$(cat "${SECRETS_KUBERNETES_DIR}/namespace")}

    # Check the API access that will be needed in the TERM signal handler
    podResponse=$(k8sCurlGet api/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/pods/$(hostname) )
    statefulSetName=$(echo ${podResponse} | jq -r '.metadata.ownerReferences[0].name')
    statefulSetPath="apis/apps/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/statefulsets/${statefulSetName}"
    statefulSetResponse=$(k8sCurlGet ${statefulSetPath} )
    isCodeForbidden=$(echo ${statefulSetResponse} | jq '.code == 403')
    if [[ ${isCodeForbidden} == "true" ]]; then
        echo "Permission error: Cannot access URL ${statefulSetPath}: $(echo ${statefulSetResponse} | jq '.reason,.code,.message')"
        exit 1
    else
        numReplicas=$(echo ${statefulSetResponse} | jq '.status.replicas')
        echo "Permissions ok: Our pod $(hostname) belongs to StatefulSet ${statefulSetName} with ${numReplicas} replicas"
    fi
fi

# Set up kubernetes node discovery
start_join_cluster=0
if env | grep "DOCKER_VERNEMQ_DISCOVERY_KUBERNETES" -q; then
    # Let's set our nodename correctly
    # https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#list-pod-v1-core
    podList=$(k8sCurlGet "api/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/pods?labelSelector=${DOCKER_VERNEMQ_KUBERNETES_LABEL_SELECTOR}")
    VERNEMQ_KUBERNETES_SUBDOMAIN=${DOCKER_VERNEMQ_KUBERNETES_SUBDOMAIN:-$(echo ${podList} | jq '.items[0].spec.subdomain' | tr '\n' '"' | sed 's/"//g')}
    if [[ $VERNEMQ_KUBERNETES_SUBDOMAIN == "null" ]]; then
        VERNEMQ_KUBERNETES_HOSTNAME=${MY_POD_NAME}.${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}.svc.${DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME}
    else
        VERNEMQ_KUBERNETES_HOSTNAME=${MY_POD_NAME}.${VERNEMQ_KUBERNETES_SUBDOMAIN}.${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}.svc.${DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME}
    fi

    sed -i.bak -r "s/VerneMQ@.+/VerneMQ@${VERNEMQ_KUBERNETES_HOSTNAME}/" ${VERNEMQ_VM_ARGS_FILE}
    # Hack into K8S DNS resolution (temporarily)
    kube_pod_names=$(echo ${podList} | jq '.items[].spec.hostname' | sed 's/"//g' | tr '\n' ' ' | sed 's/ *$//')

    for kube_pod_name in $kube_pod_names; do
        if [[ $kube_pod_name == "null" ]]; then
            echo "Kubernetes discovery selected, but no pods found. Maybe we're the first?"
            echo "Anyway, we won't attempt to join any cluster."
            break
        fi
        if [[ $kube_pod_name != $MY_POD_NAME ]]; then
            discoveryHostname="${kube_pod_name}.${VERNEMQ_KUBERNETES_SUBDOMAIN}.${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}.svc.${DOCKER_VERNEMQ_KUBERNETES_CLUSTER_NAME}"
            start_join_cluster=1
            echo "Will join an existing Kubernetes cluster with discovery node at ${discoveryHostname}"
            echo "-eval \"vmq_server_cmd:node_join('VerneMQ@${discoveryHostname}')\"" >> ${VERNEMQ_VM_ARGS_FILE}
            echo "Did I previously leave the cluster? If so, purging old state."
            curl -fsSL http://${discoveryHostname}:8888/status.json >/dev/null 2>&1 ||
                (echo "Can't download status.json, better to exit now" && exit 1)
            curl -fsSL http://${discoveryHostname}:8888/status.json | grep -q ${VERNEMQ_KUBERNETES_HOSTNAME} ||
                (echo "Cluster doesn't know about me, this means I've left previously. Purging old state..." && rm -rf /vernemq/data/*)
            break
        fi
    done
fi

if [ -f "${VERNEMQ_CONF_LOCAL_FILE}" ]; then
    cp "${VERNEMQ_CONF_LOCAL_FILE}" ${VERNEMQ_CONF_FILE}
    sed -i -r "s/###IPADDRESS###/${IP_ADDRESS}/" ${VERNEMQ_CONF_FILE}
else
    sed -i '/########## Start ##########/,/########## End ##########/d' ${VERNEMQ_CONF_FILE}

    echo "########## Start ##########" >> ${VERNEMQ_CONF_FILE}

    env | grep DOCKER_VERNEMQ | grep -v 'DISCOVERY_NODE\|KUBERNETES\|SWARM\|COMPOSE\|DOCKER_VERNEMQ_USER' | cut -c 16- | awk '{match($0,/^[A-Z0-9_]*/)}{print tolower(substr($0,RSTART,RLENGTH)) substr($0,RLENGTH+1)}' | sed 's/__/./g' >> ${VERNEMQ_CONF_FILE}

    users_are_set=$(env | grep DOCKER_VERNEMQ_USER)
    if [ ! -z "$users_are_set" ]; then
        echo "vmq_passwd.password_file = /vernemq/etc/vmq.passwd" >> ${VERNEMQ_CONF_FILE}
        touch /vernemq/etc/vmq.passwd
    fi

    for vernemq_user in $(env | grep DOCKER_VERNEMQ_USER); do
        username=$(echo $vernemq_user | awk -F '=' '{ print $1 }' | sed 's/DOCKER_VERNEMQ_USER_//g' | tr '[:upper:]' '[:lower:]')
        password=$(echo $vernemq_user | awk -F '=' '{ print $2 }')
        /vernemq/bin/vmq-passwd /vernemq/etc/vmq.passwd $username <<EOF
$password
$password
EOF
    done

    if [ -z "$DOCKER_VERNEMQ_ERLANG__DISTRIBUTION__PORT_RANGE__MINIMUM" ]; then
        echo "erlang.distribution.port_range.minimum = 9100" >> ${VERNEMQ_CONF_FILE}
    fi

    if [ -z "$DOCKER_VERNEMQ_ERLANG__DISTRIBUTION__PORT_RANGE__MAXIMUM" ]; then
        echo "erlang.distribution.port_range.maximum = 9109" >> ${VERNEMQ_CONF_FILE}
    fi

    if [ -z "$DOCKER_VERNEMQ_LISTENER__TCP__DEFAULT" ]; then
        echo "listener.tcp.default = ${IP_ADDRESS}:1883" >> ${VERNEMQ_CONF_FILE}
    fi

    if [ -z "$DOCKER_VERNEMQ_LISTENER__WS__DEFAULT" ]; then
        echo "listener.ws.default = ${IP_ADDRESS}:8080" >> ${VERNEMQ_CONF_FILE}
    fi

    if [ -z "$DOCKER_VERNEMQ_LISTENER__VMQ__CLUSTERING" ]; then
        echo "listener.vmq.clustering = ${IP_ADDRESS}:44053" >> ${VERNEMQ_CONF_FILE}
    fi

    if [ -z "$DOCKER_VERNEMQ_LISTENER__HTTP__METRICS" ]; then
        echo "listener.http.metrics = ${IP_ADDRESS}:8888" >> ${VERNEMQ_CONF_FILE}
    fi

    echo "########## End ##########" >> ${VERNEMQ_CONF_FILE}
fi

if [ ! -z "$DOCKER_VERNEMQ_ERLANG__MAX_PORTS" ]; then
    sed -i.bak -r "s/\+Q.+/\+Q ${DOCKER_VERNEMQ_ERLANG__MAX_PORTS}/" ${VERNEMQ_VM_ARGS_FILE}
fi

if [ ! -z "$DOCKER_VERNEMQ_ERLANG__PROCESS_LIMIT" ]; then
    sed -i.bak -r "s/\+P.+/\+P ${DOCKER_VERNEMQ_ERLANG__PROCESS_LIMIT}/" ${VERNEMQ_VM_ARGS_FILE}
fi

if [ ! -z "$DOCKER_VERNEMQ_ERLANG__MAX_ETS_TABLES" ]; then
    sed -i.bak -r "s/\+e.+/\+e ${DOCKER_VERNEMQ_ERLANG__MAX_ETS_TABLES}/" ${VERNEMQ_VM_ARGS_FILE}
fi

if [ ! -z "$DOCKER_VERNEMQ_ERLANG__DISTRIBUTION_BUFFER_SIZE" ]; then
    sed -i.bak -r "s/\+zdbbl.+/\+zdbbl ${DOCKER_VERNEMQ_ERLANG__DISTRIBUTION_BUFFER_SIZE}/" ${VERNEMQ_VM_ARGS_FILE}
fi

# Check configuration file
/vernemq/bin/vernemq config generate 2>&1 > /dev/null | tee /tmp/config.out | grep error

if [ $? -ne 1 ]; then
    echo "configuration error, exit"
    echo "$(cat /tmp/config.out)"
    exit $?
fi

pid=0

# SIGUSR1-handler
siguser1_handler() {
    echo "stopped"
}

# SIGTERM-handler
sigterm_handler() {
    if [ $pid -ne 0 ]; then
        if [ -d "${SECRETS_KUBERNETES_DIR}" ] ; then
            # this will stop the VerneMQ process, but first drain the node from all existing client sessions (-k)
            if [ -n "$VERNEMQ_KUBERNETES_HOSTNAME" ]; then
                terminating_node_name=VerneMQ@$VERNEMQ_KUBERNETES_HOSTNAME
            else
                terminating_node_name=VerneMQ@$IP_ADDRESS
            fi
            podList=$(k8sCurlGet "api/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/pods?labelSelector=${DOCKER_VERNEMQ_KUBERNETES_LABEL_SELECTOR}")
            kube_pod_names=$(echo ${podList} | jq '.items[].spec.hostname' | sed 's/"//g' | tr '\n' ' ' | sed 's/ *$//')
            if [ "$kube_pod_names" = "$MY_POD_NAME" ]; then
                echo "I'm the only pod remaining. Not performing leave and/or state purge."
                /vernemq/bin/vmq-admin node stop >/dev/null
            else
                # https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#read-pod-v1-core
                podResponse=$(k8sCurlGet api/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/pods/$(hostname) )
                statefulSetName=$(echo ${podResponse} | jq -r '.metadata.ownerReferences[0].name')

                # https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#-strong-read-operations-statefulset-v1-apps-strong-
                statefulSetResponse=$(k8sCurlGet "apis/apps/v1/namespaces/${DOCKER_VERNEMQ_KUBERNETES_NAMESPACE}/statefulsets/${statefulSetName}" )

                isCodeForbidden=$(echo ${statefulSetResponse} | jq '.code == 403')
                if [[ ${isCodeForbidden} == "true" ]]; then
                    echo "Permission error: Cannot access URL ${statefulSetPath}: $(echo ${statefulSetResponse} | jq '.reason,.code,.message')"
                fi

                reschedule=$(echo ${statefulSetResponse} | jq '.status.replicas == .status.readyReplicas')
                scaled_down=$(echo ${statefulSetResponse} | jq '.status.currentReplicas == .status.updatedReplicas')

                if [[ $reschedule == "true" ]]; then
                    # Perhaps is an scale down?
                    if [[ $scaled_down == "true" ]]; then
                      echo "Seems that this is a scale down scenario. Leaving cluster."
                      /vernemq/bin/vmq-admin cluster leave node=${terminating_node_name} -k && rm -rf /vernemq/data/*
                    else
                      echo "Reschedule is true. Not leaving the cluster."
                      /vernemq/bin/vmq-admin node stop >/dev/null
                    fi
                else
                    echo "Reschedule is false. Leaving the cluster."
                    /vernemq/bin/vmq-admin cluster leave node=${terminating_node_name} -k && rm -rf /vernemq/data/*
                fi
            fi
        else
            if [ -n "$DOCKER_VERNEMQ_SWARM" ]; then
                terminating_node_name=VerneMQ@$(hostname -i)
                # For Swarm we keep the old "cluster leave" approach for now
                echo "Swarm node is leaving the cluster."
                /vernemq/bin/vmq-admin cluster leave node=${terminating_node_name} -k && rm -rf /vernemq/data/*
            else
            # In non-k8s mode: Stop the vernemq node gracefully
            /vernemq/bin/vmq-admin node stop >/dev/null
            fi
        fi
        kill -s TERM ${pid}
        WAITFOR_PID=${pid}
        pid=0
        wait ${WAITFOR_PID}
    fi
    exit 143; # 128 + 15 -- SIGTERM
}

if [ ! -s ${VERNEMQ_VM_ARGS_FILE} ]; then
  echo "ls -l ${VERNEMQ_ETC_DIR}"
  ls -l ${VERNEMQ_ETC_DIR}
  echo "###" >&2
  echo "### Configuration file ${VERNEMQ_VM_ARGS_FILE} is empty! This will not work." >&2
  echo "### Exiting now." >&2
  echo "###" >&2
  exit 1
fi

# Setup OS signal handlers
trap 'siguser1_handler' SIGUSR1
trap 'sigterm_handler' SIGTERM

# Start VerneMQ
/vernemq/bin/vernemq console -noshell -noinput $@ &
pid=$!
if [ $start_join_cluster  -eq 1 ]; then
    mkdir -p /var/log/vernemq/log
    join_cluster > /var/log/vernemq/log/join_cluster.log &
fi
if [ -n "$API_KEY" ]; then
  sleep 10 && echo "Adding API_KEY..." && /vernemq/bin/vmq-admin api-key add key="${API_KEY:-DEFAULT}"
  vmq-admin api-key show
fi
wait $pid
