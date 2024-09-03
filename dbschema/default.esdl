using extension auth;

module default {
    type User {
        required identity: ext::auth::Identity {
            constraint exclusive;
        };

        required username: str {
            constraint exclusive;
        }
        required credits: int64 {
            constraint min_value(0);
        }
        image_uri: str;

        required created_at: datetime {
            readonly := true;
            default := datetime_of_statement();
        }
    }

    type Deposit {
        required credits: int64;
        required info: json;
        required remote_transaction_id: str {
            constraint exclusive;
        }

        required user: User;

        required created_at: datetime {
            readonly := true;
            default := datetime_of_statement();
        }

        index on ((.user, .created_at));
    }

    type Stream {
        required audio_uri: str;
        required audio_duration_seconds: int64;
        required credits: int64;

        required user: User;

        required created_at: datetime {
            readonly := true;
            default := datetime_of_statement();
        }
    }

    type Bid {
        required audio_uri: str;
        required audio_duration_seconds: int64;
        required credits: int64;

        required user: User;

        required created_at: datetime {
            readonly := true;
            default := datetime_of_statement();
        }

        index on ((.credits / .audio_duration_seconds, .created_at));
    }
}
