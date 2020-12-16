
export class User {
    id: number;
    name: string;
    password: string;
    email: string;
    token?: string;

    constructor(base: any) {
        this.id = base['id'];
        this.name = base['name'];
        this.password = base['password'];
        this.email = base['email'];
    }
}
