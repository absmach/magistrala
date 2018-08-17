export interface Channel {
    id?: '';
    name: string;
    connected: Thing[];
}

export interface Thing {
    id?: '';
    type: string;
    name: string;
    metadata: string;
}

export interface User {
    email: string;
    password: string;
}
