terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "4.50.0"
    }
  }
}

variable "name" {
  type = string
}

variable "project" {
  type = string
}

variable "init_script" {
  type = string
}

variable "zone" {
  type = string
}

variable "snapshot" {
  type = string
  default = ""
}

variable "machine_type" {
  type = string
  default = "e2-standard-4"
}

variable "machine_image" {
  type = string
  default = "projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20230112"
}

variable "username" {
  type = string
  default = "devpod"
}

provider "google" {
  project = var.project
  zone    = var.zone
}

resource "google_compute_instance" "workspace" {
  name         = var.name
  machine_type = var.machine_type
  zone         = var.zone

  metadata = {
    user-data = templatefile("cloud-config.yaml.tftpl", {
      username          = var.username
      init_script       = var.init_script
    })
  }

  boot_disk {
    auto_delete = false
    source      = google_compute_disk.workspace.name
  }

  network_interface {
    network = "default"
    access_config {}
  }

  desired_status = "RUNNING"

  lifecycle {
    ignore_changes = [metadata]
  }
}

resource "google_compute_disk" "workspace" {
  name  = var.name
  type  = "pd-ssd"
  zone  = var.zone
  image = var.snapshot != "" ? "" : var.machine_image
  snapshot = var.snapshot != "" ? var.snapshot : ""
  size = 50

  lifecycle {
    ignore_changes = [name, image]
  }
}

output ip_address {
  value = google_compute_instance.workspace.network_interface.0.access_config.0.nat_ip
}
