# Setup SSH config to avoid fingerprint prompts
mkdir -p ~/.ssh
cp known_hosts ~/.ssh/known_hosts
chmod 700 ~/.ssh
chmod 600 ~/.ssh/known_hosts
