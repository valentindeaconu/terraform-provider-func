/**
 * Adds two numbers together.
 *
 * Adds two numbers and returns the sum of the numbers.
 * 
 * @param {number} a - The first number.
 * @param {number} b - The second number.
 * @returns {number} The sum of `a` and `b`.
 */
$(function sum(a, b) {
  return a + b;
})

/**
 * Returns the smaller of two numbers.
 *
 * Checks which number is smaller and returns it.
 * 
 * @param {number} a - The first number.
 * @param {number} b - The second number.
 * @returns {number} The smaller of `a` and `b`.
 */
$(function min(a, b) {
  return a > b ? b : a;
})

/**
 * Concatenates two strings.
 * 
 * Same as `sum`, but for strings.
 *
 * @param {string} a - The first string.
 * @param {string} b - The second string.
 * @returns {string} The concatenated string.
 */
$(function concat(a, b) {
  return a + b;
})

/**
 * Merges two arrays by concatenating them.
 *
 * Same as `sum`, but for arrays.
 * 
 * @param {string[]} a - The first array.
 * @param {string[]} b - The second array.
 * @returns {string[]} A new array containing elements of `a` followed by elements of `b`.
 */
$(function extend(a, b) {
  return a.concat(b);
})

/**
 * Creates a Person object.
 *
 * Uses values received as parameters and returns an object with those values as fields.
 * 
 * @param {string} name - The name of the person.
 * @param {number} age - The age of the person.
 * @returns {{name: string; age: number;}} An object with two attributes, name and age.
 */
$(function create_object(name, age) {
  return { name, age };
})
