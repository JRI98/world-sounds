CREATE MIGRATION m1ddaaf74rfiilizvru5vdgaxicmhryflw6xh6bgmrlhwnw7sabhqq
    ONTO initial
{
  CREATE EXTENSION pgcrypto VERSION '1.3';
  CREATE EXTENSION auth VERSION '1.0';
  CREATE TYPE default::User {
      CREATE REQUIRED LINK identity: ext::auth::Identity {
          CREATE CONSTRAINT std::exclusive;
      };
      CREATE REQUIRED PROPERTY created_at: std::datetime {
          SET default := (std::datetime_of_statement());
          SET readonly := true;
      };
      CREATE REQUIRED PROPERTY credits: std::int64 {
          CREATE CONSTRAINT std::min_value(0);
      };
      CREATE PROPERTY image_uri: std::str;
      CREATE REQUIRED PROPERTY username: std::str {
          CREATE CONSTRAINT std::exclusive;
      };
  };
  CREATE TYPE default::Bid {
      CREATE REQUIRED PROPERTY audio_duration_seconds: std::int64;
      CREATE REQUIRED PROPERTY created_at: std::datetime {
          SET default := (std::datetime_of_statement());
          SET readonly := true;
      };
      CREATE REQUIRED PROPERTY credits: std::int64;
      CREATE INDEX ON (((.credits / .audio_duration_seconds), .created_at));
      CREATE REQUIRED LINK user: default::User;
      CREATE REQUIRED PROPERTY audio_uri: std::str;
  };
  CREATE TYPE default::Deposit {
      CREATE REQUIRED LINK user: default::User;
      CREATE REQUIRED PROPERTY created_at: std::datetime {
          SET default := (std::datetime_of_statement());
          SET readonly := true;
      };
      CREATE INDEX ON ((.user, .created_at));
      CREATE REQUIRED PROPERTY credits: std::int64;
      CREATE REQUIRED PROPERTY info: std::json;
      CREATE REQUIRED PROPERTY remote_transaction_id: std::str {
          CREATE CONSTRAINT std::exclusive;
      };
  };
  CREATE TYPE default::Stream {
      CREATE REQUIRED LINK user: default::User;
      CREATE REQUIRED PROPERTY audio_duration_seconds: std::int64;
      CREATE REQUIRED PROPERTY audio_uri: std::str;
      CREATE REQUIRED PROPERTY created_at: std::datetime {
          SET default := (std::datetime_of_statement());
          SET readonly := true;
      };
      CREATE REQUIRED PROPERTY credits: std::int64;
  };
};
