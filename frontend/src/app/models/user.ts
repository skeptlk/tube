
export class User {
    id: number;
    name: string;
    password: string;
    email: string;
    token?: string;
    isAdmin: string;
    createdAt: Date;

    constructor(base: any) {
        this.id = base['id'];
        this.name = base['name'];
        this.password = base['password'];
        this.email = base['email'];
        this.isAdmin = base['isAdmin'];
        this.createdAt = base['createdAt'];
    }
}
