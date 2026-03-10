provider "aws" {}
provider "test" {}

resource "aws_instance" "web" {
    lifecycle {
        action_trigger {
            events = [before_create]
            actions = [action.test_action.verb]
        }
    }
}

action "test_action" "verb" {}