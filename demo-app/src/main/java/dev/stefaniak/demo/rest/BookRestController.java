// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package com.example.demo.rest;

import com.example.demo.service.BookService;
import java.util.List;
import java.util.stream.Collectors;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookRestController {
  private BookService service;

  @ResponseStatus(value = HttpStatus.NOT_FOUND)
  public class ResourceNotFoundException extends RuntimeException {
  }
  public BookRestController(BookService service) {
    this.service = service;
  }

  @GetMapping("/books")
  public List<Book> books() {
    return service.getAll().stream()
        .map(b -> new Book(b.getId(), b.getTitle(), b.getAuthor(), b.getCategory()))
        .collect(Collectors.toList());
  }

  @GetMapping("/book/{id}")
  public Book book(@PathVariable Long id) {
    return service.get(id)
        .map(b -> new Book(b.getId(), b.getTitle(), b.getAuthor(), b.getCategory()))
        .orElseThrow(ResourceNotFoundException::new);
  }
}
