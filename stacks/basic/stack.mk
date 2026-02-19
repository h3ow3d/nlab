.PHONY: basic basic-destroy

basic: STACK_NAME=basic
basic: NETWORK_NAME=basic_net
basic: NETWORK_XML=stacks/basic/network.xml
basic-destroy: STACK_NAME=basic
basic-destroy: NETWORK_NAME=basic_net

basic: ## Stand up the basic stack (attacker + target VMs on basic_net)
	mkdir -p logs
	./scripts/generate-key.sh $(STACK_NAME)
	./scripts/create-network.sh $(NETWORK_XML) $(NETWORK_NAME) $(STACK_NAME)
	./scripts/create-dashboard.sh $(STACK_NAME) $(NETWORK_NAME) & \
	  DASH_PID=$$!; \
	  trap 'kill $$DASH_PID 2>/dev/null || true; exit 130' INT TERM; \
	  ./scripts/create-vm.sh $(STACK_NAME) attacker 4096 2 $(NETWORK_NAME) > logs/attacker.log 2>&1 & \
	  ATT_PID=$$!; \
	  ./scripts/create-vm.sh $(STACK_NAME) target 2048 2 $(NETWORK_NAME) > logs/target.log 2>&1 & \
	  TGT_PID=$$!; \
	  wait $$ATT_PID $$TGT_PID; \
	  kill $$DASH_PID 2>/dev/null || true; \
	  wait $$DASH_PID 2>/dev/null || true
	./scripts/launch-tmux.sh $(STACK_NAME) $(NETWORK_NAME)

basic-destroy: ## Tear down the basic stack
	./scripts/destroy-vm.sh $(STACK_NAME) attacker
	./scripts/destroy-vm.sh $(STACK_NAME) target
	./scripts/destroy-network.sh $(NETWORK_NAME)
