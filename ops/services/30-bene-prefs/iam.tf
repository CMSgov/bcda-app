
data "aws_iam_policy_document" "assume_bucket_role" {
  statement {
    sid       = "AssumeBucketRole"
    actions   = ["sts:AssumeRole"]
    resources = [module.platform.ssm.bene-prefs.iam_bucket_role_arn.value]
  }
}

data "aws_iam_policy_document" "subscribe_bfd_topic" {
  statement {
    sid       = "AllowBFDSNSSubscribe"
    effect    = "Allow"
    actions   = ["sns:Subscribe"]
    resources = [module.platform.ssm.bene-prefs.sns_topic_arn.value]
  }
}

data "aws_iam_policy_document" "bucket_sqs" {
  statement {
    sid = "SqsReceiveDeleteMessages"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:GetQueueAttributes",
      "sqs:DeleteMessage",
    ]
    resources = [aws_sqs_queue.this.arn]
  }
}