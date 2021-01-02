import { Component, OnInit } from '@angular/core';
import { User } from '../../models';
import { AdminService } from './../../services';

@Component({
    selector: 'admin-users-list',
    templateUrl: './admin-users-list.component.html',
})
export class AdminUsersListComponent implements OnInit {

    users: User[] = [];
    pages: number = 1;
    currentPage: number = 0;
    total: number;
    pageSize: number = 10;

    constructor(private adminService: AdminService) 
    { }

    ngOnInit(): void {
        this.adminService
            .getUsers(0, this.pageSize)
            .subscribe(resp => { 
                this.users = resp.users; 
                this.total = resp.total;
                this.pages = Math.ceil(resp.total / this.pageSize);
            });
    }

    switchPage(page: number): void {
        const limit = this.pageSize;
        this.currentPage = page;
        this.adminService
            .getUsers(page*limit, limit)
            .subscribe(resp => { this.users = resp.users; });
    }

    deleteUser(id: number) {
        this.adminService.deleteUser(id).subscribe(() => {
            this.switchPage(this.currentPage); // reload
        })

    }

}
