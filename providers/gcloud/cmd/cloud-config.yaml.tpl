#cloud-config
users:
  - name: devpod
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    shell: /bin/bash
write_files:
  - path: /opt/devpod/init
    permissions: "0755"
    encoding: b64
    content: {{ .InitScript }}
  - path: /etc/systemd/system/devpod.service
    permissions: "0644"
    content: |
      [Unit]
      Description=DevPod

      [Service]
      User=devpod
      ExecStart=/opt/devpod/init
      Restart=always
      RestartSec=10

      [Install]
      WantedBy=multi-user.target
runcmd:
  - chown devpod:devpod /home/devpod
  - systemctl daemon-reload
  - systemctl enable devpod
  - systemctl start devpod