resource "aws_efs_file_system" "efs" {
  creation_token   = var.name
  performance_mode = var.performance_mode
  encrypted        = "true"
  kms_key_id       = var.kms_key_id

  tags = {
    Name    = var.name
    env     = var.env
    app     = var.app
    service = var.service
  }
  lifecycle {
    ignore_changes = [
      throughput_mode,
    ]
  }
}

resource "aws_efs_mount_target" "efs" {
  count           = length(var.subnets)
  file_system_id  = aws_efs_file_system.efs.id
  subnet_id       = element(var.subnets, count.index)
  security_groups = flatten([aws_security_group.efs.id])
}

resource "aws_security_group" "efs" {
  name = "${var.name}-efs"
  // "NFS" in this context refers to the ability to mount the filesystem
  // over port 2049. See https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/4/html/reference_guide/ch-nfs.
  description = "Allow NFS traffic."
  vpc_id      = var.vpc_id

  lifecycle {
    create_before_destroy = true
  }

  ingress {
    from_port = "2049"
    to_port   = "2049"
    protocol  = "tcp"

    security_groups = var.additional_ingress_sgs
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name    = var.name
    env     = var.env
    app     = var.app
    service = var.service
  }
}

