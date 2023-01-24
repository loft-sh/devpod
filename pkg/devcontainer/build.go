package devcontainer

func Build() {
	// docker buildx build --load --build-arg BUILDKIT_INLINE_CACHE=1 -f /tmp/devcontainercli-root/container-features/0.29.0-1674845358112/Dockerfile-with-features -t vsc-test-38392bab732ee88d03bc36cc66c16189 --target dev_containers_target_stage --build-arg VARIANT=3 --build-arg _DEV_CONTAINERS_BASE_IMAGE=dev_container_auto_added_stage_label /home/devpod/devpod/workspace/test
}
