export interface Channel {
    id?: '';
    name: string;
    connected: Client[];
}

export interface Client {
    id?: '';
    type: string;
    name: string;
    payload: string;
}

export interface User {
    email: string;
    password: string;
}
