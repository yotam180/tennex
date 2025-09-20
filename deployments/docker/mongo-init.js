// MongoDB Initialization Script for Tennex
// This script sets up the database and creates necessary users and indexes

// Switch to the tennex database
db = db.getSiblingDB('tennex');

// Create application user
db.createUser({
    user: 'tennex_app',
    pwd: 'tennex_app_password',
    roles: [
        {
            role: 'readWrite',
            db: 'tennex'
        }
    ]
});

// Create indexes for client_sessions collection
db.client_sessions.createIndexes([
    // Unique index on session_id
    { 
        "session_id": 1 
    },
    // Compound index for client queries
    { 
        "client_id": 1, 
        "created_at": -1 
    },
    // Index for status queries
    { 
        "status": 1 
    },
    // Index for expiration cleanup
    { 
        "expires_at": 1 
    },
    // Sparse index for WhatsApp JID lookups
    { 
        "whatsapp_jid": 1 
    }
]);

// Create sample collections with validation (optional)
db.createCollection("client_sessions", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["client_id", "session_id", "status", "created_at", "expires_at"],
            properties: {
                client_id: {
                    bsonType: "string",
                    description: "Client identifier - required"
                },
                session_id: {
                    bsonType: "string",
                    description: "Unique session identifier - required"
                },
                whatsapp_jid: {
                    bsonType: "string",
                    description: "WhatsApp JID after successful connection"
                },
                status: {
                    enum: ["waiting_for_scan", "connected", "disconnected", "expired"],
                    description: "Session status - required"
                },
                session_data: {
                    bsonType: "binData",
                    description: "Encrypted session data blob"
                },
                created_at: {
                    bsonType: "date",
                    description: "Session creation timestamp - required"
                },
                connected_at: {
                    bsonType: "date",
                    description: "Connection timestamp"
                },
                disconnected_at: {
                    bsonType: "date",
                    description: "Disconnection timestamp"
                },
                expires_at: {
                    bsonType: "date",
                    description: "Session expiration timestamp - required"
                },
                last_seen: {
                    bsonType: "date",
                    description: "Last activity timestamp"
                }
            }
        }
    }
});

// Create events collection for future use
db.createCollection("events", {
    validator: {
        $jsonSchema: {
            bsonType: "object",
            required: ["id", "timestamp", "type", "convo_id", "device_id", "account_id"],
            properties: {
                id: {
                    bsonType: "string",
                    description: "Event unique identifier - required"
                },
                seq: {
                    bsonType: "long",
                    description: "Sequence number for ordering"
                },
                timestamp: {
                    bsonType: "date",
                    description: "Event timestamp - required"
                },
                type: {
                    enum: ["msg_in", "msg_out", "edit", "delete", "reaction", "read", "delivery", "contact", "thread_meta", "media_key", "presence", "connection"],
                    description: "Event type - required"
                },
                convo_id: {
                    bsonType: "string",
                    description: "Conversation identifier - required"
                },
                wa_message_id: {
                    bsonType: "string",
                    description: "WhatsApp message ID"
                },
                sender_jid: {
                    bsonType: "string",
                    description: "Sender WhatsApp JID"
                },
                payload: {
                    bsonType: "object",
                    description: "Event payload data"
                },
                attachment_ref: {
                    bsonType: "string",
                    description: "Reference to attachment storage"
                },
                device_id: {
                    bsonType: "string",
                    description: "Device identifier - required"
                },
                account_id: {
                    bsonType: "string",
                    description: "Account identifier - required"
                }
            }
        }
    }
});

// Create indexes for events collection
db.events.createIndexes([
    // Unique index on event id
    { 
        "id": 1 
    },
    // Index for sequence-based queries
    { 
        "seq": 1 
    },
    // Compound index for conversation queries
    { 
        "convo_id": 1, 
        "timestamp": -1 
    },
    // Index for message ID lookups
    { 
        "wa_message_id": 1 
    },
    // Index for sender queries
    { 
        "sender_jid": 1, 
        "timestamp": -1 
    },
    // Index for event type queries
    { 
        "type": 1, 
        "timestamp": -1 
    },
    // Compound index for account-based queries
    { 
        "account_id": 1, 
        "device_id": 1, 
        "timestamp": -1 
    }
]);

print("MongoDB initialization completed successfully!");
print("Collections created: client_sessions, events");
print("Indexes created for optimal query performance");
print("Application user 'tennex_app' created with readWrite permissions");
