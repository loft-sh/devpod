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

variable "zone" {
  type = string
}

provider "google" {
  project = var.project
  zone    = var.zone
}

resource "google_compute_snapshot" "snapshot" {
  name        = var.name
  source_disk = var.name
  zone        = var.zone
}
