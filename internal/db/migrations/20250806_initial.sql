CREATE TABLE urls (
        shorturl VARCHAR(7) PRIMARY KEY,
        url VARCHAR(100) NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        expires_at TIMESTAMP DEFAULT NULL,
        active BOOLEAN DEFAULT TRUE

);

CREATE TABLE analytics (
        id SERIAL PRIMARY KEY,
        shorturl VARCHAR(7) NOT NULL,
        count INTEGER DEFAULT 0,
        ip_address VARCHAR(50) DEFAULT NULL,
        user_agent VARCHAR(1024) DEFAULT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

        FOREIGN KEY (shorturl) REFERENCES urls(shorturl)

);
