.PHONY: template template-destroy

STACK_NAME=template
NETWORK_NAME=template_net
NETWORK_XML=stacks/template/network.xml

template:
	mkdir -p logs
	./scripts/generate-key.sh $(STACK_NAME)
	./scripts/create-network.sh $(NETWORK_XML) $(NETWORK_NAME)
	./scripts/create-vm.sh $(STACK_NAME) attacker 4096 2 $(NETWORK_NAME) > logs/attacker.log 2>&1 &
	./scripts/create-vm.sh $(STACK_NAME) target 2048 2 $(NETWORK_NAME) > logs/target.log 2>&1 &
	wait
	./scripts/launch-tmux.sh $(STACK_NAME) $(NETWORK_NAME)

template-destroy:
	./scripts/destroy-vm.sh $(STACK_NAME) attacker
	./scripts/destroy-vm.sh $(STACK_NAME) target
	./scripts/destroy-network.sh $(NETWORK_NAME)
