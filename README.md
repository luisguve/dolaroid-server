# Root "/"

Check whether the user is logged in and return its data:

- username
- user id 
- type of account

And whether the client should send its coords.

Method: **GET**

credentials: include

## Response for user not logged in:
```json
{
  "isLoggedIn": false,
  "sendLocation": false
}
```

## Response for user logged in:
```json
{
  "isLoggedIn": true,
  "session": {
    "username": "username",
    "typeOfAccount": "regular/business/admin",
    "userId": "user-id"
  },
  "sendLocation": true
}
```
`sendLocation` might be `false`.

# Login "/login"

Validate user credentials and return it's data: username, user id and type of account and wether the client should send its coords.

Method: **POST**

credentials: include

## Body:
```json
{
  "username": "username",
  "password": "password"
}
```

## Response for successful login:
```json
{
  "isLoggedIn": true,
  "session": {
    "username": "username",
    "typeOfAccount": "regular/business/admin",
    "userId": "user id"
  },
  "sendLocation": true
}
```
`sendLocation` might be `false`.

Error responses:

- Username does not exist: "User unregistered"
- Stored password and submitted password don't match: "Invalid username or password"

# Signup "/signup"

Register user if username is available and return it's data:

- username
- user id
- type of account

And wether the client should send its coords.

Method: **POST**

credentials: include

## Body:
```json
{
  "username": "username",
  "password": "password"
}
```

Response for successful signup:
```json
{
  "isLoggedIn": true,
  "session": {
    "username": "username",
    "typeOfAccount": "regular/business/admin",
    "userId": "user id"
  },
  "sendLocation": true
}
```
`sendLocation` might be `false`.

Error responses:
- Username already exists: "Username " + username + " already taken"

# Logout "/logout"

Remove session cookie

Method: **POST**

credentials: include

## body: empty

Response for successful logout: 200 OK

Error responses:

- No session cookie or successful logout response:

# Submit coords "/location"

Send coords of client, store them in session cookie and return 200 OK.

Method: **POST**

credentials: include

## Body:
```json
{
  "latt": "6.42375",
  "longt": "-66.58973"
}
```
Error response:

Sent empty or invalid coords: status 400 bad request.

# Get review "/review"

Method: GET

credentials: include

## Query parameters:

- **sn** for serial number
- **value** denomination
- **series** series year

Example: `http://localhost:8000/review?sn=44SOMETHING12&value=100&series=2013`

## Full response

If the user is logged in, it returns a full response:
```json
{
  "billInfo": {
    "serialNumber": "44 SOMETHING 12",
    "value": "100",
    "series": "2013"
  },
  "goodReviews": 0,
  "badReviews": 0,
  "avgRating": 0.0,
  "defects": [],
  "userReviews": {
    "goodReviews": [
      {
        "userId": "user-id",
        "date": "js date",
        "comment": "comment",
        "rating": 5,
        "defects": [],
        "location": {
          "latt": "6.42375",
          "longt": "-66.58973",
          "city": "city",
          "region": "region",
          "country": "country"
        }
      }
    ],
    "badReviews": [
      {
        "userId": "user-id",
        "date": "js date",
        "comment": "comment",
        "rating": 0,
        "defects": ["ft-3d-ribbon", "ft-watermark"],
        "location": {
          "latt": "6.42375",
          "longt": "-66.58973",
          "city": "city",
          "region": "region",
          "country": "country"
        }
      }
    ]
  },
  "businessReviews": {
    "goodReviews": [
      {
        "userId": "user-id",
        "date": "js date",
        "comment": "comment",
        "rating": 5,
        "defects": [],
        "location": {
          "latt": "6.42375",
          "longt": "-66.58973",
          "city": "city",
          "region": "region",
          "country": "country"
        }
      }
    ],
    "badReviews": [
      {
        "userId": "user-id",
        "date": "js date",
        "comment": "comment",
        "rating": 0,
        "defects": ["ft-3d-ribbon", "ft-watermark"],
        "location": {
          "latt": "6.42375",
          "longt": "-66.58973",
          "city": "city",
          "region": "region",
          "country": "country"
        }
      }
    ]
  },
  "details": {
    "in": {
      "date": "js date",
      "involved": "from/to",
      "subject": "details subject",
      "notes": "private notes"
    },
    "out": {
      "date": "js date",
      "involved": "from/to",
      "subject": "details subject",
      "notes": "private notes"
    },
  }
}
```

`defects` is an array of strings, in case of a bad review.

`rating` is a number between 1 and 5 in case of a good review.

## Basic response

If the user is not logged in, this is what the response looks like
```json
{
  "billInfo": {
    "serialNumber": "44 SOMETHING 12",
    "value": "100",
    "series": "2013"
  },
  "goodReviews": 0,
  "badReviews": 0,
  "avgRating": 0.0
}
```

# Post review "/review"

Method: **POST**

credentials: include

## Body:
```json
{
  "billInfo": {
    "serialNumber": "44 SOMETHING 12",
    "value": "100",
    "series": "2013"
  },
  "review": {
    "date": "js date",
    "comment": "comment",
    "rating": 0,
    "defects": [],
  },
  "details": {
    "typeOfDetail": "incoming/outgoing",
    "detailsData": {
      "date": "js date",
      "involved": "from/to",
      "subject": "details subject",
      "notes": "private notes"
    }
  }
}
```

`defects` is an array of strings, in case of a bad review.

`rating` is a number between 1 and 5 in case of a good review.

`details` is optional

Response for successful post: 200 OK

Error responses:.

- The session cookie does not have the coords: "Location required"
- The user is not logged in: 401 Unauthorized