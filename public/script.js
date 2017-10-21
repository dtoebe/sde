const docID = location.pathname.match(/\/documents\/(.*)/)[1];
const host = 'localhost:3030';
const doc = document.getElementById("sendText");
const title = document.getElementById("docTitle");
const moConf = {
    attributes: true,
    childList: true,
    characterData: true,
    subtree: true
};

let sock = new WebSocket(getURL("ws", host, "editor", docID));
let sessID = '';
let observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
        sendData(mutation);
    });
});

function getURL(proto, host, path, id) {
    return `${proto}://${host}/${path}/${id}`;
}

function getDoc(e) {
    if (e.target.readyState === 4) {
        if (e.target.status === 200) {
            const data = JSON.parse(e.target.responseText);
            if (data.body) {
                observer.disconnect();
                doc.innerHTML = data.body;
                observer.observe(doc, moConf);
            }
            if (data.title) {
                title.innerText = data.title;
            }
        }
    }
}

function sendData(e) {
    let data = JSON.stringify({
        sender: sessID,
        body: doc.innerHTML
    });

    console.log("SEND", data);
    sock.send(data);
}

let xhr = new XMLHttpRequest();
xhr.onreadystatechange = getDoc;
xhr.open("GET", getURL("http", host, "docbody", docID));
xhr.send();

sock.onopen = (e) => {
    console.log("CONNECTED");
};

sock.onclose = (e) => {
    console.log("CLOSED");
};

sock.onmessage = (e) => {
    console.log("RECEIVE:", e.data);

    let data = JSON.parse(e.data);
    if (data.id) {
        sessID = data.id;
        return;
    }

    if (data.sender) {
        if (data.sender === sessID) {
            return;
        }

        observer.disconnect();
        doc.innerHTML = decodeURI(data.body);
        observer.observe(doc, moConf);
    }
};

observer.observe(doc, moConf);